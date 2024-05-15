package internal

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/neorpc"
	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
)

// RPCClient used in integration test.
type RPCClient struct {
	addr []string
	len  int32
	inc  atomic.Int32
	// The only txSender's duty is to send `sendrawtransaction` requests in
	// order not to affect bench results by sending service requests via the
	// same connection. txSender has different fasthttp settings than blockRequester.
	txSender *fasthttp.Client
	// blockRequester should do the rest of work, e.g. fetch blocks count, fetch
	// blocks and etc.
	blockRequester *fasthttp.Client

	timeout time.Duration
}

// DefaultTimeout used for requests.
const DefaultTimeout = time.Second * 30

var (
	// ErrMempoolOOM is returned from `sendrawtransaction` when node cannot process transaction due to mempool OOM.
	ErrMempoolOOM = errors.New("node cannot process transaction due to mempool OOM")
)

// NewRPCClient creates new client for RPC communications.
func NewRPCClient(v *viper.Viper, maxConnsPerHost int) *RPCClient {
	var addresses []string
	for _, addr := range v.GetStringSlice("rpcAddress") {
		if addr == "" {
			continue
		}

		addresses = append(addresses, "http://"+addr)
	}

	buf := make([]byte, 8)
	if _, err := crand.Read(buf); err != nil {
		log.Fatal("could not initialize randomizer for round robin")
	}

	src := binary.BigEndian.Uint64(buf)
	rand.NewSource(int64(src))

	rand.Shuffle(len(addresses), func(i, j int) {
		addresses[i], addresses[j] = addresses[j], addresses[i]
	})

	timeout := DefaultTimeout
	if v := v.GetDuration("request_timeout"); v >= 0 {
		timeout = v
	}

	txSender := &fasthttp.Client{
		MaxIdemponentCallAttempts: 1, // don't repeat queries
		ReadTimeout:               timeout,
		WriteTimeout:              timeout,
		MaxConnsPerHost:           maxConnsPerHost,
	}

	blockRequester := &fasthttp.Client{
		MaxIdemponentCallAttempts: 1, // don't repeat queries
		ReadTimeout:               timeout,
		WriteTimeout:              timeout,
		MaxConnsPerHost:           2, // let's keep it small in order not to overload the nodes by open service connections in `Workers` mode
	}

	c := &RPCClient{
		txSender:       txSender,
		blockRequester: blockRequester,
		addr:           addresses,
		len:            int32(len(addresses)),

		timeout: timeout,
	}
	c.inc.Store(rand.Int31())
	return c
}

// GetLastBlock returns last block from blockchain.
func (c *RPCClient) GetLastBlock(ctx context.Context) (*block.Block, error) {
	num, err := c.GetBlockCount(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetBlock(ctx, num-1)
}

func (c *RPCClient) GetVersion(ctx context.Context) (*result.Version, error) {
	res := new(result.Version)
	rpc := `{ "jsonrpc": "2.0", "id": 1, "method": "getversion", "params": [] }`
	if err := c.doRPCCall(ctx, rpc, res, c.blockRequester); err != nil {
		return res, err
	}

	return res, nil
}

// SendTX sends transaction.
func (c *RPCClient) SendTX(ctx context.Context, tx string) error {
	var res struct {
		Hash util.Uint256 `json:"hash"`
	}
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "sendrawtransaction", "params": ["%s"]}`, tx)

	if err := c.doRPCCall(ctx, rpc, &res, c.txSender); err != nil {
		msg := err.Error()
		if errors.Is(err, neorpc.ErrMempoolCapReached) || strings.Contains(msg, "OutOfMemory") {
			return ErrMempoolOOM
		}
		return err
	} else if res.Hash.Equals(util.Uint256{}) {
		return errors.New("SendTX request failed")
	}

	return nil
}

// GetBlock sends getblock RPC request.
func (c *RPCClient) GetBlock(ctx context.Context, index int) (*block.Block, error) {
	res := ""
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "getblock", "params": [%v]}`, index)
	if err := c.doRPCCall(ctx, rpc, &res, c.blockRequester); err != nil {
		return nil, err
	}

	blk := block.New(false)
	body, err := base64.StdEncoding.DecodeString(res)
	if err != nil {
		return nil, err
	}

	rd := io.NewBinReaderFromBuf(body)
	blk.DecodeBinary(rd)

	return blk, rd.Err
}

// GetBlockCount send getblockcount RPC request.
func (c *RPCClient) GetBlockCount(ctx context.Context) (int, error) {
	num := 0
	rpc := `{"jsonrpc": "2.0", "id": 1, "method": "getblockcount", "params": []}`
	return num, c.doRPCCall(ctx, rpc, &num, c.blockRequester)
}

func (c *RPCClient) doRPCCall(_ context.Context, call string, result interface{}, client *fasthttp.Client) error {
	idx := c.inc.Add(1) % c.len

	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()
	req.SetBodyString(call)
	req.SetRequestURI(c.addr[idx])
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set(fasthttp.HeaderContentType, "application/json; charset=utf-8")

	// dump request for debug reasons only:
	// reqData, _ := httputil.DumpRequest(req, true)
	// fmt.Println(string(reqData))

	resp := new(neorpc.Response)
	if err := client.Do(req, res); err != nil {
		return fmt.Errorf("error after calling rpc server %w", err)
	} else if body, code := res.Body(), res.StatusCode(); code != fasthttp.StatusOK && len(body) == 0 {
		return fmt.Errorf("http error: %d %s", code, res.String())
	} else if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("could not unmarshal response body: %q %w", string(body), err)
	} else if resp.Error != nil && resp.Error.Code != 0 {
		return resp.Error
	} else if err = json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("could not unmarshal result body: %q %w", string(body), err)
	}
	return nil
}
