package internal

import (
	"io"
)

type (
	empty int

	// BenchMode can be wrk and rate.
	BenchMode string
)

const (
	// DevNull is ioutil.Discard but for io.Reader.
	DevNull = empty(0)

	// ModeWorker runs the specific number of workers.
	ModeWorker = BenchMode("wrk")

	// ModeRate runs the specific requests rate limit.
	ModeRate = BenchMode("rate")
)

// String returns string form of BenchMode.
func (m BenchMode) String() string { return string(m) }

// Read returns zero and io.EOF.
func (empty) Read([]byte) (int, error) { return 0, io.EOF }
