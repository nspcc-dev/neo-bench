package internal

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
)

type (
	// rpcResponse is base JSON RPC response
	// easyjson:json
	rpcResponse struct {
		ID      int             `json:"id"`
		Version string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		*errorResponse
	}

	// errorResponse struct for RPC error response.
	// easyjson:json
	errorResponse struct {
		ErrorResult struct {
			Code    int    `json:"code"`
			Data    string `json:"data"`
			Message string `json:"message"`
		} `json:"error"`
	}

	// versionResponse struct for RPC version response.
	// easyjson:json
	versionResponse struct {
		Port    int    `json:"port"`
		Nonce   int    `json:"nonce"`
		Version string `json:"useragent"`
	}

	// RPCClient used in integration test.
	RPCClient struct {
		addr []string
		len  int32
		inc  *atomic.Int32
		// The only txSender's duty is to send `sendrawtransaction` requests in
		// order not to affect bench results by sending service requests via the
		// same connection. txSender has different fasthttp settings than blockRequester.
		txSender *fasthttp.Client
		// blockRequester should do the rest of work, e.g. fetch blocks count, fetch
		// blocks and etc.
		blockRequester *fasthttp.Client

		timeout time.Duration
	}
)

// DefaultTimeout used for requests.
const DefaultTimeout = time.Second * 30

var reg = regexp.MustCompile(`[^\w.-]+`)

func (v errorResponse) Error() string {
	if v.ErrorResult.Code == 0 {
		return ""
	}

	return "Error #" + strconv.Itoa(v.ErrorResult.Code) + ": " + v.ErrorResult.Message + " " + v.ErrorResult.Data
}

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

	return &RPCClient{
		txSender:       txSender,
		blockRequester: blockRequester,
		addr:           addresses,
		len:            int32(len(addresses)),
		inc:            atomic.NewInt32(rand.Int31()),

		timeout: timeout,
	}
}

// GetLastBlock returns last block from blockchain.
func (c *RPCClient) GetLastBlock(ctx context.Context) (*block.Block, error) {
	num, err := c.GetBlockCount(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetBlock(ctx, num-1)
}

func (c *RPCClient) GetVersion(ctx context.Context) (string, error) {
	res := new(versionResponse)
	rpc := `{ "jsonrpc": "2.0", "id": 1, "method": "getversion", "params": [] }`
	if err := c.doRPCCall(ctx, rpc, res, c.blockRequester); err != nil {
		return "", err
	}

	return strings.Trim(reg.ReplaceAllString(res.Version, "_"), "_"), nil
}

// SendTX sends transaction.
func (c *RPCClient) SendTX(ctx context.Context, tx string) error {
	var res struct {
		Hash util.Uint256 `json:"hash"`
	}
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "sendrawtransaction", "params": ["%s"]}`, tx)

	if err := c.doRPCCall(ctx, rpc, &res, c.txSender); err != nil {
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

	blk := block.New(netmode.PrivNet)
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
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "getblockcount", "params": []}`)
	return num, c.doRPCCall(ctx, rpc, &num, c.blockRequester)
}

func (c *RPCClient) doRPCCall(_ context.Context, call string, result interface{}, client *fasthttp.Client) error {
	idx := c.inc.Inc() % c.len

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

	resp := new(rpcResponse)
	if err := client.Do(req, res); err != nil {
		return errors.Errorf("error after calling rpc server %s", err)
	} else if body, code := res.Body(), res.StatusCode(); code != fasthttp.StatusOK && len(body) == 0 {
		return errors.Errorf("http error: %d %s", code, res.String())
	} else if err := json.Unmarshal(body, &resp); err != nil {
		return errors.Errorf("could not unmarshal response body: %q %v", string(body), err)
	} else if resp.errorResponse != nil && resp.ErrorResult.Code != 0 {
		return resp
	} else if err = json.Unmarshal(resp.Result, result); err != nil {
		return errors.Errorf("could not unmarshal result body: %q %v", string(body), err)
	}
	return nil
}
