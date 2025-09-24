package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/moby/moby/client"
)

type (
	dockerStateParams struct {
		enableLogger bool
		criteria     []string
		period       time.Duration
		verifier     func([]container.Summary) error
	}

	// DockerStater interface.
	DockerStater interface {
		Run(ctx context.Context, cb StatCallback)
		Update(ctx context.Context) (cpu, mem float64, err error)
	}

	dockerState struct {
		*log.Logger

		per time.Duration
		cli *client.Client
		cnr []container.Summary
	}

	// StatOption is an option type to configure docker state.
	StatOption func(*dockerStateParams)

	// StatCallback used to report current docker container resource usage.
	StatCallback func(cpu, mem float64)
)

// StatCriteria sets criteria to select containers.
func StatCriteria(criteria []string) StatOption {
	return func(p *dockerStateParams) {
		p.criteria = criteria
	}
}

// StatEnableLogger enables logs.
func StatEnableLogger() StatOption {
	return func(p *dockerStateParams) {
		p.enableLogger = true
	}
}

// StatPeriod sets period for resource usage fetching.
func StatPeriod(dur time.Duration) StatOption {
	return func(p *dockerStateParams) {
		p.period = dur
	}
}

// StatListVerifier sets containers list verifier.
func StatListVerifier(verifier func([]container.Summary) error) StatOption {
	return func(p *dockerStateParams) {
		if p.verifier == nil {
			return
		}

		p.verifier = verifier
	}
}

// NewStats creates new DockerStater, to fetch resource usage by containers.
func NewStats(ctx context.Context, opts ...StatOption) (DockerStater, error) {
	p := &dockerStateParams{
		verifier: func(_ []container.Summary) error { return nil },
	}

	for i := range opts {
		opts[i](p)
	}

	cli, err := client.NewClientWithOpts(client.WithVersion("1.40")) // version mey need to be downgraded on different hosts
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}

	criteria := filters.NewArgs()
	for i := range p.criteria {
		criteria.Add("label", p.criteria[i])
	}

	list, err := cli.ContainerList(ctx, container.ListOptions{Filters: criteria})
	if err != nil {
		return nil, err
	}

	if err = p.verifier(list); err != nil {
		return nil, err
	}

	ds := &dockerState{
		cnr:    list,
		cli:    cli,
		per:    p.period,
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	if !p.enableLogger {
		ds.SetOutput(io.Discard)
	}

	if _, _, err = ds.Update(ctx); err != nil {
		return nil, err
	}

	return ds, nil
}

// Update returns current resource usage by containers.
func (s *dockerState) Update(ctx context.Context) (cpu, mem float64, err error) {
	var (
		result container.StatsResponseReader
		stats  container.StatsResponse
	)

	for i := range s.cnr {
		id := s.cnr[i].ID

		if result, err = s.cli.ContainerStats(ctx, id, false); err != nil {
			cpu = 0
			mem = 0
			return
		}

		if err = json.NewDecoder(result.Body).Decode(&stats); err != nil {
			cpu = 0
			mem = 0
			return
		}

		// Update state
		curCPU, curMem := usage(&stats)

		cpu += curCPU
		mem += curMem
	}

	return
}

// Run worker to periodically fetching resource usage by containers.
func (s *dockerState) Run(ctx context.Context, cb StatCallback) {
	done := ctx.Done()
	tick := time.NewTimer(s.per)

loop:
	for {
		select {
		case <-done:
			break loop
		case <-tick.C:
			cpu, mem, err := s.Update(ctx)
			if errors.Is(err, context.Canceled) {
				break loop
			} else if err != nil {
				s.Printf("Something went wrong: %v", err)
			}

			cb(cpu, mem)

			tick.Reset(s.per)
		}
	}
}

func usage(s *container.StatsResponse) (cpu, mem float64) {
	var (
		systemDelta = float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
		cpuDelta    = float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
	)

	mem = float64(s.MemoryStats.Usage)

	if cache, ok := s.MemoryStats.Stats["cache"]; ok {
		mem -= float64(cache)
	}

	mem = mem / 1024 / 1024

	if systemDelta > 0 && cpuDelta > 0 {
		cpu = (cpuDelta / systemDelta) * 100
	}

	return
}
