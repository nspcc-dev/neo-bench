package internal

import (
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"
)

type (
	reporter struct {
		*sync.Mutex

		name       string
		TxCount    int32
		ErrCount   int32
		AverageRPS float64
		TPS        []tpsInfo
		TPSPool    []tpsInfo
		Stats      [][3]float64 // MillisecondsFromStart, CPU, Mem
	}

	// tpsInfo stores information useful for counting TPS
	tpsInfo struct {
		// DeltaTime is a time in milliseconds since the previous block timestamp
		DeltaTime uint64
		// TxCount is the number of transactions in block
		TxCount int
		TPS     float64
	}

	// Reporter interface.
	Reporter interface {
		io.WriterTo
		UpdateErr(v int32)
		UpdateCnt(v int32)
		UpdateRPS(v float64)
		UpdateTPS(deltaTime uint64, txCount int, v float64)
		UpdateRes(start time.Time, cpu, mem float64)
	}

	reportParams struct {
		description string
		mode        BenchMode
		wrkLimit    int
		rateLimit   int
		timeLimit   time.Duration
	}

	// ReportOption is an option type to configure reporter.
	ReportOption func(*reportParams)
)

// ReportMode sets report mode.
func ReportMode(mode BenchMode) ReportOption {
	return func(p *reportParams) {
		p.mode = mode
	}
}

// ReportDescription sets description for current report.
func ReportDescription(description string) ReportOption {
	return func(p *reportParams) {
		p.description = description
	}
}

// ReportTimeLimit sets time limit for reporter.
func ReportTimeLimit(limit time.Duration) ReportOption {
	return func(p *reportParams) {
		p.timeLimit = limit
	}
}

// ReportWorkersCount sets count of workers for current report.
func ReportWorkersCount(cnt int) ReportOption {
	return func(p *reportParams) {
		p.wrkLimit = cnt
	}
}

// ReportRate sets rate limit for current report.
func ReportRate(rate int) ReportOption {
	return func(p *reportParams) {
		p.rateLimit = rate
	}
}

// NewReporter creates reporter.
func NewReporter(opts ...ReportOption) Reporter {
	p := reportParams{
		description: "unknown",
		mode:        "unknown",
		wrkLimit:    -1,
		timeLimit:   -1,
	}

	for i := range opts {
		opts[i](&p)
	}

	var count int
	switch p.mode {
	case ModeWorker:
		count = p.wrkLimit
	case ModeRate:
		count = p.rateLimit
	}
	return &reporter{
		Mutex: new(sync.Mutex),
		name:  fmt.Sprintf("%s / %d %s / %s", p.description, count, p.mode, p.timeLimit),
	}
}

// WriteTo writes report to io.Writer.
func (r *reporter) WriteTo(rw io.Writer) (int64, error) {
	r.Lock()
	defer r.Unlock()

	out := io.MultiWriter(rw, os.Stdout)

	tps := .0
	for i := range r.TPS {
		tps += r.TPS[i].TPS
	}

	cpu := .0
	for i := range r.Stats {
		cpu += r.Stats[i][1]
	}

	mem := .0
	for i := range r.Stats {
		mem += r.Stats[i][2]
	}

	var (
		num int
		cnt int64
		err error

		tpsCount = float64(len(r.TPS))
		resCount = float64(len(r.Stats))
		errRate  = float64(r.ErrCount*100) / float64(r.TxCount+r.ErrCount)
	)

	if num, err = fmt.Fprintf(out, "%s\n\n", r.name); err != nil {
		return int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintf(out, "TXs ≈ %d\n", r.TxCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintf(out, "RPS ≈ %0.3f\n", r.AverageRPS); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintf(out, "RPC Errors  ≈ %d / %0.3f%%\n", r.ErrCount, errRate); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if num, err = fmt.Fprintf(out, "TPS ≈ %0.3f\n\n", tps/tpsCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if num, err = fmt.Fprintf(out, "CPU ≈ %0.3f%%\n", cpu/resCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if num, err = fmt.Fprintf(out, "Mem ≈ %0.3fMB\n\n", mem/resCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintln(out, "MillisecondsFromStart, CPU, Mem"); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)
	for i := range r.Stats {
		if num, err = fmt.Fprintf(out, "%0.3f, %0.3f%%, %0.3fMB\n", r.Stats[i][0], r.Stats[i][1], r.Stats[i][2]); err != nil {
			return cnt + int64(num), err
		}
		cnt += int64(num)
	}

	if num, err = fmt.Fprintln(out, "\nDeltaTime, TransactionsCount, TPS"); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	for i := range r.TPS {
		if num, err = fmt.Fprintf(out, "%d, %d, %0.3f\n", r.TPS[i].DeltaTime, r.TPS[i].TxCount, r.TPS[i].TPS); err != nil {
			return cnt + int64(num), err
		}
		cnt += int64(num)
	}

	return cnt, nil
}

// UpdateRPS sets current average rps rate.
func (r *reporter) UpdateRPS(v float64) {
	if v <= 0 || math.IsNaN(v) {
		return
	}

	r.Lock()
	defer r.Unlock()

	r.AverageRPS = v
}

// UpdateCnt sets count of sent txs.
func (r *reporter) UpdateCnt(v int32) {
	if v <= 0 {
		return
	}

	r.Lock()
	defer r.Unlock()

	r.TxCount = v
}

// UpdateErr sets errors count while send TX to RPC.
func (r *reporter) UpdateErr(v int32) {
	if v <= 0 {
		return
	}

	r.Lock()
	defer r.Unlock()

	r.ErrCount = v
}

// UpdateTPS sets current tps rate
func (r *reporter) UpdateTPS(deltaTime uint64, txCount int, v float64) {
	if v < 0 || math.IsNaN(v) {
		return
	}

	r.Lock()
	defer r.Unlock()

	if v > 0 {
		r.TPS = append(r.TPS, r.TPSPool...)
		r.TPS = append(r.TPS, tpsInfo{
			DeltaTime: deltaTime,
			TxCount:   txCount,
			TPS:       v,
		})
		r.TPSPool = nil
	} else {
		r.TPSPool = append(r.TPSPool, tpsInfo{
			DeltaTime: deltaTime,
			TxCount:   txCount,
			TPS:       v,
		})
	}
}

// UpdateRes sets current resource usage by containers.
func (r *reporter) UpdateRes(start time.Time, cpu, mem float64) {
	r.Lock()
	defer r.Unlock()

	r.Stats = append(r.Stats, [3]float64{float64(time.Now().Sub(start).Nanoseconds()) / 1000000, cpu, mem})
}
