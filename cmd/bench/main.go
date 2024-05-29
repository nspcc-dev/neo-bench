package main

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/nspcc-dev/neo-bench/internal"
)

// Main steps for testing are:
// - prepare docker image with neo-go node or c# node
// - start privatenet
// - start docker nodeContainerReq
// - create RPC client
// - generate input for RPC client
// - start sending txes to the node
// - measure how much TX could be sent

func main() {
	v := internal.InitSettings()

	log.Printf("Used %s rpc addresses", "["+strings.Join(v.GetStringSlice("rpcAddress"), ", ")+"]")

	ctx, cancel := context.WithCancel(internal.NewGracefulContext())
	defer cancel()

	var (
		workers         = v.GetInt("workers")
		rate            int
		msPerBlock      int
		mempoolOOMDelay time.Duration
		threshold       time.Duration
		dump            *internal.Dump
		desc            = v.GetString("desc")
		timeLimit       = v.GetDuration("timeLimit")
		mode            = internal.BenchMode(v.GetString("mode"))
		client          *internal.RPCClient
		disableStats    = v.GetBool("disable-stats")
	)

	if mode == internal.ModeRate {
		rate = v.GetInt("rateLimit")
		threshold = time.Duration(time.Second.Nanoseconds() / int64(rate) * int64(workers))
	}

	client = internal.NewRPCClient(v, workers)
	version, err := client.GetVersion(ctx)
	if err != nil {
		log.Fatalf("could not receive RPC Node version: %v", err)
	}
	msPerBlock = version.Protocol.MillisecondsPerBlock
	if msPerBlock > 1000 {
		mempoolOOMDelay = time.Duration(msPerBlock) * time.Millisecond / 50
	} else {
		mempoolOOMDelay = time.Duration(msPerBlock) * time.Millisecond / 10
	}

	reg := regexp.MustCompile(`[^\w.-]+`)
	versionStr := strings.Trim(reg.ReplaceAllString(version.UserAgent, "_"), "_")
	log.Println("Run benchmark for " + desc + " :: " + versionStr)

	//raising the limits. Some performance gains were achieved with the + workers count (not a lot).
	runtime.GOMAXPROCS(runtime.NumCPU() + workers)

	rep := internal.NewReporter(
		internal.ReportMode(mode),
		internal.ReportDescription(desc+" :: "+versionStr),
		internal.ReportTimeLimit(timeLimit),
		internal.ReportWorkersCount(workers),
		internal.ReportRate(rate),
		internal.ReportDefaultMSPerBlock(msPerBlock))

	out, err := os.Create(v.GetString("out"))
	if err != nil {
		log.Fatalf("could not open report: %v", err)
	}

	defer func() {
		log.Println("try to write profile")
		if _, err := rep.WriteTo(out); err != nil {
			log.Fatalf("could not write result: %v", err)
		}

		if err := out.Close(); err != nil {
			log.Fatalf("could not close report: %v", err)
		}
	}()
	if !disableStats {
		statsPeriod := time.Second

		ds, err := internal.NewStats(ctx,
			internal.StatEnableLogger(),
			internal.StatPeriod(statsPeriod),
			internal.StatCriteria([]string{"stats"}),
			internal.StatListVerifier(func(list []types.Container) error {
				if len(list) == 0 {
					return errors.New("containers not found by criteria")
				}

				return nil
			}))

		if err != nil {
			log.Fatalf("could not create docker stats grabber: %v", err)
		}

		statsStart := time.Now()
		// Run stats worker:
		go ds.Run(ctx, func(cpu, mem float64) {
			rep.UpdateRes(statsStart, cpu, mem)
			log.Printf("CPU: %0.3f%%, Mem: %0.3fMB", cpu, mem)
		})
	}

	if in := v.GetString("in"); in != "" {
		dump = internal.ReadDump(in)
	} else {
		log.Fatalf("Transactions dump file wasn't specified.")
	}

	wrk, err := internal.NewWorkers(
		internal.WorkerDump(dump),
		internal.WorkerMode(mode),
		internal.WorkersCount(workers),
		internal.Rate(rate),
		internal.WorkerStopper(cancel),
		internal.WorkerTimeLimit(timeLimit),
		internal.WorkerThreshold(threshold),
		internal.WorkerBlockchainClient(client),
		internal.WorkerMempoolOOMDelay(mempoolOOMDelay),
		internal.WorkerRPSReporter(rep.UpdateRPS),
		internal.WorkerTPSReporter(rep.UpdateTPS),
		internal.WorkerErrReporter(rep.UpdateErr),
		internal.WorkerCntReporter(rep.UpdateCnt),
	)

	if err != nil {
		log.Println(err)
		return
	}

	wrk.Prepare(ctx, v.GetBool("vote"), dump.BenchOptions)

	log.Printf("fetch current block count")
	blk, err := client.GetLastBlock(ctx)
	if err != nil {
		log.Fatalf("could not fetch last block: %v", err)
	}

	log.Println("Waiting for an empty block to be processed")
	startBlockIndex := blk.Index
	// 10*msPerBlock attempts (need some more time for mixed consensus)
	for attempt := 0; attempt < 40; attempt++ {
		blk, err = client.GetLastBlock(ctx)
		if err != nil {
			log.Fatalf("could not fetch last block: %v", err)
		}
		if blk.Index > startBlockIndex {
			break
		}
		time.Sleep(time.Duration(msPerBlock) * time.Millisecond / 4)
	}
	if blk.Index == startBlockIndex {
		log.Fatalf("Timeout waiting for a new empty block")
	}

	log.Printf("Started test from block = %v at unix time = %v", blk.Index, blk.Timestamp)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go wrk.Parser(ctx, blk)
	go wrk.Sender(ctx)

	wrk.Wait()
}
