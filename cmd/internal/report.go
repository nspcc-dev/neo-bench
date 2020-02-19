package internal

import (
	"fmt"
	"io"
	"math"
	"sync"
	"time"
)

type (
	reporter struct {
		*sync.Mutex

		name  string
		RPS   []float64
		TPS   []float64
		Stats [][2]float64 // CPU, Mem
	}

	// Reporter interface.
	Reporter interface {
		io.WriterTo
		UpdateRPS(v float64)
		UpdateTPS(v float64)
		UpdateRes(cpu, mem float64)
	}

	reportParams struct {
		name      string
		mode      BenchMode
		wrkLimit  int
		timeLimit time.Duration
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

// ReportName sets description (name) for current report.
func ReportName(name string) ReportOption {
	return func(p *reportParams) {
		p.name = name
	}
}

// ReportTimeLimit sets time limit for reporter.
func ReportTimeLimit(limit time.Duration) ReportOption {
	return func(p *reportParams) {
		p.timeLimit = limit
	}
}

// ReportWorkersCount sets count of workers / rate limit for current report.
func ReportWorkersCount(cnt int) ReportOption {
	return func(p *reportParams) {
		p.wrkLimit = cnt
	}
}

// NewReporter creates reporter.
func NewReporter(opts ...ReportOption) Reporter {
	p := reportParams{
		name:      "unknown",
		mode:      "unknown",
		wrkLimit:  -1,
		timeLimit: -1,
	}

	for i := range opts {
		opts[i](&p)
	}

	return &reporter{
		Mutex: new(sync.Mutex),
		name:  fmt.Sprintf("%s / %d %s / %s", p.name, p.wrkLimit, p.mode, p.timeLimit),
	}
}

// WriteTo writes report to io.Writer.
func (r *reporter) WriteTo(rw io.Writer) (int64, error) {
	r.Lock()
	defer r.Unlock()

	rps := 0.0
	for i := range r.RPS {
		rps += r.RPS[i]
	}

	tps := 0.0
	for i := range r.TPS {
		tps += r.TPS[i]
	}

	var (
		num int
		cnt int64
		err error

		rpsCount = float64(len(r.RPS))
		tpsCount = float64(len(r.TPS))
	)

	if num, err = fmt.Fprintf(rw, "\n%s\n", r.name); err != nil {
		return int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintf(rw, "\nRPS ≈ %0.3f\n", rps/rpsCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if num, err = fmt.Fprintf(rw, "\nTPS ≈ %0.3f\n", tps/tpsCount); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	if _, err := fmt.Fprintln(rw, "\nCPU, Mem"); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)
	for i := range r.Stats {
		if num, err = fmt.Fprintf(rw, "%0.3f, %0.3f\n", r.Stats[i][0], r.Stats[i][1]); err != nil {
			return cnt + int64(num), err
		}
		cnt += int64(num)
	}

	if num, err = fmt.Fprintln(rw, "\nTPS"); err != nil {
		return cnt + int64(num), err
	}
	cnt += int64(num)

	for i := range r.TPS {
		if num, err = fmt.Fprintf(rw, "%0.3f\n", r.TPS[i]); err != nil {
			return cnt + int64(num), err
		}
		cnt += int64(num)
	}

	return cnt, nil
}

// UpdateRPS sets current rps rate.
func (r *reporter) UpdateRPS(v float64) {
	if v <= 0 || math.IsNaN(v) {
		return
	}

	r.Lock()
	defer r.Unlock()

	r.RPS = append(r.RPS, v)
}

// UpdateTPS sets current tps rate
func (r *reporter) UpdateTPS(v float64) {
	if v <= 0 || math.IsNaN(v) {
		return
	}

	r.Lock()
	defer r.Unlock()

	r.TPS = append(r.TPS, v)
}

// UpdateRes sets current resource usage by containers.
func (r *reporter) UpdateRes(cpu, mem float64) {
	r.Lock()
	defer r.Unlock()

	r.Stats = append(r.Stats, [2]float64{cpu, mem})
}
