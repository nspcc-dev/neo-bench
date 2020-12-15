package internal

import (
	"context"
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/core/block"
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
		parsed       chan struct{}
		sentOut      chan struct{}
		waiter       *sync.WaitGroup
		countTxs     *atomic.Int32 // stores count of completed queries
		countErr     *atomic.Int32
		hasStarted   *atomic.Bool
		parsedCount  int
		parsedBlocks map[int]struct{}
	}

	doerParams struct {
		wrkCount    int
		cli         *RPCClient
		mode        BenchMode
		rate        int
		threshold   time.Duration
		timeLimit   time.Duration
		dump        *Dump
		cntReporter func(cnt int32)
		errReporter func(cnt int32)
		rpsReporter func(rps float64)
		tpsReporter func(deltaTime uint64, txCount int, tps float64)
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

// Rate sets the number requests per second.
func Rate(rate int) WorkerOption {
	return func(p *doerParams) {
		p.rate = rate
	}
}

// WorkerDump sets dump of transactions that would be used for sending requests and parse blocks.
func WorkerDump(dump *Dump) WorkerOption {
	return func(p *doerParams) {
		p.dump = dump
	}
}

// WorkerRPSReporter sets method that would be used to report current RPS.
func WorkerRPSReporter(reporter func(v float64)) WorkerOption {
	return func(p *doerParams) {
		// ignore empty func
		if reporter == nil {
			return
		}

		p.rpsReporter = reporter
	}
}

// WorkerTPSReporter sets method that would be used to report current TPS.
func WorkerTPSReporter(reporter func(deltaTime uint64, txCount int, v float64)) WorkerOption {
	return func(p *doerParams) {
		// ignore empty func
		if reporter == nil {
			return
		}

		p.tpsReporter = reporter
	}
}

// WorkerErrReporter sets method that would be used to report errors count while send TX to RPC.
func WorkerErrReporter(reporter func(v int32)) WorkerOption {
	return func(p *doerParams) {
		// ignore empty func
		if reporter == nil {
			return
		}

		p.errReporter = reporter
	}
}

// WorkerCntReporter sets method that would be used to report count of Tx's sent to RPC.
func WorkerCntReporter(reporter func(v int32)) WorkerOption {
	return func(p *doerParams) {
		// ignore empty func
		if reporter == nil {
			return
		}

		p.cntReporter = reporter
	}
}

// NewWorkers creates new worker manager.
func NewWorkers(opts ...WorkerOption) (Worker, error) {
	p := doerParams{
		// set defaults:
		cntReporter: func(_ int32) {},
		errReporter: func(_ int32) {},
		rpsReporter: func(_ float64) {},
		tpsReporter: func(_ uint64, _ int, _ float64) {},
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
		log.Printf("Init worker with %d QPS / %s time limit (%d txs will try to send)", p.rate, p.timeLimit, ln)
	case ModeWorker:
		log.Printf("Init %d workers / %s time limit (%d txs will try to send)", p.wrkCount, p.timeLimit, ln)
	}

	w := &doer{
		doerParams:   p,
		txCount:      ln,
		Mutex:        new(sync.Mutex),
		waiter:       new(sync.WaitGroup),
		parsed:       make(chan struct{}),
		sentOut:      make(chan struct{}),
		countTxs:     atomic.NewInt32(0),
		countErr:     atomic.NewInt32(0),
		hasStarted:   atomic.NewBool(false),
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

loop:
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

			if err := d.cli.SendTX(ctx, d.dump.Transactions[i]); err != nil {
				d.countErr.Inc()
				continue loop
				//d.stop()
				//return
			}

			since := time.Since(start)
			count := d.countTxs.Inc()
			d.rpsReporter(float64(count) / since.Seconds())

			if d.threshold > 0 {
				waitFor := start.Add(time.Duration(d.threshold.Nanoseconds() * (i + 1))).Sub(time.Now())
				if waitFor > 0 {
					time.Sleep(waitFor)
				}
			}
		}
	}
}

// Wait waits when all workers stop.
func (d *doer) Wait() {
	// wait until all request workers stopped
	d.waiter.Wait()
	log.Println("all request workers stopped")

	// wait until sender worker is done
	<-d.sentOut
	log.Println("sender worker stopped")

	// wait until parser worker is done
	<-d.parsed
	log.Println("parser worker stopped")
}

// Parser worker that periodically fetch blocks and parse them.
func (d *doer) Parser(ctx context.Context, blk *block.Block) {
	defer close(d.parsed)

	done := ctx.Done()
	period := time.Second
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

			if int32(d.parsedCount) >= d.countTxs.Load() {
				select {
				case <-d.sentOut:
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
}

func (d *doer) parse(ctx context.Context, startBlock int, lastTime *uint64) (lastBlock int) {
	var (
		cnt int
		err error
		tps float64
		blk *block.Block

		parsedCount int
	)

	lastBlock, err = d.cli.GetBlockCount(ctx)
	if err != nil {
		log.Printf("could not fetch block count: %v", err)
		d.stop()
		return
	}

	ln := d.countTxs.Load() - int32(lastBlock-startBlock)
	if ln < 0 {
		ln = 0
	}

	// log.Printf("%d txs left to parse", ln)

	for i := startBlock; i < lastBlock; i++ {
		parsedCount = 0

		if _, ok := d.parsedBlocks[i]; !ok {

			d.parsedBlocks[i] = struct{}{}
			if blk, err = d.cli.GetBlock(ctx, i); err != nil {
				log.Printf("could not get block: %v", err)
				continue
			}

			if cnt = len(blk.Transactions); cnt < 1 {
				log.Printf("empty block: %d", i)
			} else if !d.hasStarted.Load() {
				d.hasStarted.Store(true)
			}

			// Timestamp is in milliseconds so we multiply numerator by 1000 to be more precise.
			dt := blk.Timestamp - *lastTime
			if tps = float64(cnt) * 1000 / float64(dt); math.IsNaN(tps) || tps < 0 {
				tps = 0
			}

			// update last block timestamp
			*lastTime = blk.Timestamp

			// do not add zero TPS in case if there were no non-empty blocks yet
			if tps == 0 {
				if !d.hasStarted.Load() {
					continue
				}
			}

			// report current tps
			d.tpsReporter(dt, len(blk.Transactions), tps)

			for i := 0; i < cnt; i++ {
				tx := blk.Transactions[i]
				if len(tx.Scripts) > 0 {
					if _, ok := d.dump.Hashes[tx.Hash().String()]; ok {
						parsedCount++
					}
				}
			}

			d.parsedCount += parsedCount
			log.Printf("(#%d/%d) %d Tx's in %d ms %f tps", i, parsedCount, cnt, dt, tps)
		}
	}

	return
}

// Sender worker that sends requests to the RPC server.
func (d *doer) Sender(ctx context.Context) {
	defer close(d.sentOut)

	idx := atomic.NewInt64(0)

	start := time.Now()

	for i := 0; i < d.wrkCount; i++ {
		go d.worker(ctx, idx, start)
	}

	d.waiter.Wait()

	since := time.Since(start)
	count := d.countTxs.Load()
	errCount := d.countErr.Load()

	log.Printf("Sent %d transactions in %s", count, since)
	log.Printf("RPS: %5.3f", float64(count)/since.Seconds())

	d.cntReporter(count)
	d.errReporter(errCount)
	d.rpsReporter(float64(count) / since.Seconds())

	if errCount == 0 {
		log.Println("All transactions have been sent successfully")
	}

	log.Printf("RPC Errors: %d / %0.3f%%", errCount, (float64(errCount)/float64(count+errCount))*100)
}
