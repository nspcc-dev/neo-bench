package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// NewGracefulContext returns graceful context
func NewGracefulContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		sig := <-ch
		fmt.Println()
		log.Printf("received signal: %s", sig.String())
		cancel()
	}()
	return ctx
}
