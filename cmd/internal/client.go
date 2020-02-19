package internal

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/CityOfZion/neo-go/pkg/core/block"
	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
)

// ErrorResponse struct for testing.
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Data    string `json:"data"`
		Message string `json:"message"`
	} `json:"error"`
}

// SendTXResponse struct for testing.
type SendTXResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  bool   `json:"result"`
	ID      int    `json:"id"`
	ErrorResponse
}

// GetBlockCountResponse struct for testing.
type GetBlockCountResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  int    `json:"result"`
	ID      int    `json:"id"`
}

// GetBlockResponse struct for testing.
type GetBlockResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
	ID      int    `json:"id"`
}

// RPCClient used in integration test.
type RPCClient struct {
	addr []string
	len  int32
	inc  *atomic.Int32
	cli  *fasthttp.Client

	timeout time.Duration
}

// DefaultTimeout used for requests.
const DefaultTimeout = time.Second * 30

func (e ErrorResponse) String() string {
	return "Error #" + strconv.Itoa(e.Error.Code) + ": " + e.Error.Message + " " + e.Error.Data
}

// NewRPCClient creates new client for RPC communications.
func NewRPCClient(v *viper.Viper) *RPCClient {
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

	cli := &fasthttp.Client{
		MaxIdemponentCallAttempts: 1, // don't repeat queries
		ReadTimeout:               timeout,
		WriteTimeout:              timeout,
		MaxConnsPerHost:           5_000,
	}

	return &RPCClient{
		cli:  cli,
		addr: addresses,
		len:  int32(len(addresses)),
		inc:  atomic.NewInt32(rand.Int31()),

		timeout: timeout,
	}
}

// makes a new RPC client and calls node by RPC.
// getBlockCount returns current block index.
func getBlockCount(ctx context.Context, client *RPCClient) (int, error) {
	bodyBlockCount := client.GetBlockCount(ctx)
	var respBlockCount GetBlockCountResponse
	err := json.Unmarshal(bodyBlockCount, &respBlockCount)
	if err != nil {
		return 0, errors.Errorf("could not unmarshal block count: %#v", err)
	}
	return respBlockCount.Result, nil
}

// getBlock returns block by index.
func getBlock(ctx context.Context, client *RPCClient, index int) (*block.Block, error) {
	bodyBlock := client.GetBlock(ctx, index)
	var respBlock GetBlockResponse

	if err := json.Unmarshal(bodyBlock, &respBlock); err != nil {
		return nil, errors.Errorf("could not unmarshal block: %#v", err)
	}
	decodedResp, _ := hex.DecodeString(respBlock.Result)

	blk := new(block.Block)
	newReader := io.NewBinReaderFromBuf(decodedResp)
	blk.DecodeBinary(newReader)
	return blk, nil
}

// GetLastBlock returns last block from blockchain.
func (c *RPCClient) GetLastBlock(ctx context.Context) (*block.Block, error) {
	num, err := getBlockCount(ctx, c)
	if err != nil {
		return nil, err
	}

	return getBlock(ctx, c, num-1)
}

// SendTX sends transaction.
func (c *RPCClient) SendTX(ctx context.Context, tx string) []byte {
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "sendrawtransaction", "params": ["%s"]}`, tx)
	return c.doRPCCall(ctx, rpc)
}

// GetBlock sends getblock RPC request.
func (c *RPCClient) GetBlock(ctx context.Context, index int) []byte {
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "getblock", "params": [%v]}`, index)
	return c.doRPCCall(ctx, rpc)
}

// GetBlockCount send getblockcount RPC request.
func (c *RPCClient) GetBlockCount(ctx context.Context) []byte {
	rpc := fmt.Sprintf(`{"jsonrpc": "2.0", "id": 1, "method": "getblockcount", "params": []}`)
	return c.doRPCCall(ctx, rpc)
}

func (c *RPCClient) doRPCCall(_ context.Context, rpcCall string) []byte {
	idx := c.inc.Inc() % c.len

	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()
	req.SetBodyString(rpcCall)
	req.SetRequestURI(c.addr[idx])
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set(fasthttp.HeaderContentType, "application/json; charset=utf-8")

	// dump request for debug reasons only:
	// reqData, _ := httputil.DumpRequest(req, true)
	// fmt.Println(string(reqData))

	if err := c.cli.Do(req, res); err != nil {
		log.Fatalf("error after calling rpc server %s", err)
	}

	return bytes.TrimSpace(res.Body())
}
