package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/CityOfZion/neo-go/pkg/core/block"
	"go.uber.org/atomic"
)

type (
	// Worker interface.
	Worker interface {
		Wait()
		Sender(ctx context.Context)
		Parser(ctx context.Context, block *block.Block)
	}

	doer struct {
		doerParams
		*sync.Mutex
		txCount      int
		finished     chan struct{}
		waiter       *sync.WaitGroup
		counter      *atomic.Int32 // stores count of completed queries
		parsedCount  int
		parsedBlocks map[int]struct{}
	}

	doerParams struct {
		wrkCount    int
		cli         *RPCClient
		mode        BenchMode
		threshold   time.Duration
		timeLimit   time.Duration
		dump        *Dump
		rpsReporter func(rps float64)
		tpsReporter func(tps float64)
		stop        context.CancelFunc
	}

	// WorkerOption is an option type to configure workers.
	WorkerOption func(*doerParams)
)

// WorkerMode sets the specific benchmark mode.
func WorkerMode(mode BenchMode) WorkerOption {
	return func(p *doerParams) {
		p.mode = mode
	}
}

// WorkerStopper sets context.CancelFunc.
func WorkerStopper(stop context.CancelFunc) WorkerOption {
	return func(p *doerParams) {
		p.stop = stop
	}
}

// WorkerBlockchainClient sets blockchain client.
func WorkerBlockchainClient(cli *RPCClient) WorkerOption {
	return func(p *doerParams) {
		p.cli = cli
	}
}

// WorkerTimeLimit sets time limit to send requests.
func WorkerTimeLimit(limit time.Duration) WorkerOption {
	return func(p *doerParams) {
		p.timeLimit = limit
	}
}

// WorkerThreshold sets delay between requests for the specific worker.
func WorkerThreshold(threshold time.Duration) WorkerOption {
	return func(p *doerParams) {
		p.threshold = threshold
	}
}

// WorkersCount sets the specific number of workers that would be run.
func WorkersCount(cnt int) WorkerOption {
	return func(p *doerParams) {
		p.wrkCount = cnt
	}
}

// WorkerDump sets dump of transactions that would be used for sending requests and parse blocks.
func WorkerDump(dump *Dump) WorkerOption {
	return func(p *doerParams) {
		p.dump = dump
	}
}

// WorkerRPSReporter sets method that would be used for report current RPS.
func WorkerRPSReporter(reporter func(v float64)) WorkerOption {
	return func(p *doerParams) {
		p.rpsReporter = reporter
	}
}

// WorkerTPSReporter sets method that would be used for report current TPS.
func WorkerTPSReporter(reporter func(v float64)) WorkerOption {
	return func(p *doerParams) {
		p.tpsReporter = reporter
	}
}

// NewWorkers creates new worker manager.
func NewWorkers(opts ...WorkerOption) (Worker, error) {
	p := doerParams{
		// set defaults:
		rpsReporter: func(_ float64) {},
		tpsReporter: func(_ float64) {},
		stop:        func() { log.Fatal("default stopper") },
	}

	for i := range opts {
		opts[i](&p)
	}

	switch {
	case p.wrkCount < 1:
		return nil, errors.New("workers count could not be empty")
	case p.dump == nil:
		return nil, errors.New("dump could not be empty")
	case len(p.dump.Transactions) < 1:
		return nil, errors.New("txs could not be empty")
	case len(p.dump.Hashes) < 1:
		return nil, errors.New("hashes could not be empty")
	case p.cli == nil:
		return nil, errors.New("blockchain client count could not be empty")
	}

	ln := len(p.dump.Transactions)

	switch p.mode {
	case ModeRate:
		log.Printf("Init worker with %d QPS / %s time limit (%d txs will try to send)", p.wrkCount, p.timeLimit, ln)
	case ModeWorker:
		log.Printf("Init %d workers / %s time limit (%d txs will try to send)", p.wrkCount, p.timeLimit, ln)
	}

	w := &doer{
		doerParams:   p,
		txCount:      ln,
		Mutex:        new(sync.Mutex),
		waiter:       new(sync.WaitGroup),
		finished:     make(chan struct{}),
		counter:      atomic.NewInt32(0),
		parsedBlocks: make(map[int]struct{}),
	}

	w.waiter.Add(w.wrkCount)

	return w, nil
}

func (d *doer) worker(ctx context.Context, idx *atomic.Int64, start time.Time) {
	var (
		done  = ctx.Done()
		ln    = int64(d.txCount)
		timer = time.NewTimer(d.timeLimit)
	)

	defer func() {
		timer.Stop()
		d.waiter.Done()
	}()

	for {
		select {
		case <-done:
			return
		case <-timer.C:
			return
		default:
			i := idx.Inc()
			if i >= ln {
				return
			}

			body := d.cli.SendTX(ctx, d.dump.Transactions[i])
			var resp SendTXResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				log.Printf("ERROR !! Response json: %v %q", err, string(body))
				d.stop()
				return
			} else if !resp.Result {
				log.Println(string(body))
				log.Printf("ERROR !! Request failed: %s", resp)
				d.stop()
				return
			}

			since := time.Since(start)
			count := d.counter.Inc()
			d.rpsReporter(float64(count) / since.Seconds())

			if d.threshold > 0 {
				time.Sleep(d.threshold)
			}
		}
	}
}

// Wait waits when all workers stop.
func (d *doer) Wait() { d.waiter.Wait() }

// Parser worker that periodically fetch blocks and parse them.
func (d *doer) Parser(ctx context.Context, blk *block.Block) {
	done := ctx.Done()
	period := 5 * time.Second
	ticker := time.NewTimer(period)
	timeout := time.NewTimer(d.timeLimit + 5*time.Minute)
	lastBlockIndx := int(blk.Index)
	lastBlockTime := blk.Timestamp

loop:
	for {
		select {
		case <-done:
			break loop
		case <-timeout.C:
			log.Println("time limit for parsing blocks exceeded...")
			break loop
		case <-ticker.C:
			// parse new blocks:
			lastBlockIndx = d.parse(ctx, lastBlockIndx, &lastBlockTime)

			// reset timer:
			ticker.Reset(period)

			if int32(d.parsedCount) >= d.counter.Load() {
				select {
				case <-d.finished:
					break loop
				default:
					// not finished yet..
				}
			}
		}
	}

	// run parse before end:
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	d.parse(ctx, lastBlockIndx, &lastBlockTime)

	log.Printf("Sent %v transactions in %v seconds", d.counter.Load(), lastBlockTime-blk.Timestamp)
}

func (d *doer) parse(ctx context.Context, startBlock int, lastTime *uint32) (lastBlock int) {
	var (
		cnt int
		err error
		tps float64
		blk *block.Block

		parsedCount int
	)

	lastBlock, err = getBlockCount(ctx, d.cli)
	if err != nil {
		log.Printf("could not fetch block count: %v", err)
		d.stop()
		return
	}

	ln := d.counter.Load() - int32(lastBlock-startBlock)
	if ln < 0 {
		ln = 0
	}

	// log.Printf("%d txs left to parse", ln)

	for i := startBlock; i < lastBlock; i++ {
		parsedCount = 0

		if _, ok := d.parsedBlocks[i]; !ok {

			d.parsedBlocks[i] = struct{}{}
			if blk, err = getBlock(ctx, d.cli, i); err != nil {
				log.Printf("could not get block: %v", err)
				continue
			}

			if cnt = len(blk.Transactions); cnt <= 1 {
				log.Printf("empty block: %d", i)
				continue
			}

			dt := blk.Timestamp - *lastTime
			if tps = float64(cnt-1) / float64(dt); math.IsNaN(tps) || tps < 0 {
				tps = 0
			}

			// update last block timestamp
			*lastTime = blk.Timestamp

			// report current tps
			d.tpsReporter(tps)

			for i := 1; i < cnt; i++ {
				tx := blk.Transactions[i]
				if len(tx.Scripts) > 0 {
					if _, ok := d.dump.Hashes[tx.Hash().String()]; ok {
						parsedCount++
						d.parsedCount++
					}
				}
			}

			log.Printf("(#%d/%d) %d Tx's in %d secs %f tps", i, parsedCount, cnt, dt, tps)
		}
	}

	return
}

// Sender worker that sends requests to the RPC server.
func (d *doer) Sender(ctx context.Context) {
	idx := atomic.NewInt64(0)

	start := time.Now()

	for i := 0; i < d.wrkCount; i++ {
		go d.worker(ctx, idx, start)
	}

	d.waiter.Wait()

	since := time.Since(start)
	count := d.counter.Load()

	close(d.finished)

	log.Printf("Sended %d txs for %s", count, since)
	log.Printf("RPS: %5.3f", float64(count)/since.Seconds())

	d.rpsReporter(float64(count) / since.Seconds())

	log.Println("All transactions were sent")
}
