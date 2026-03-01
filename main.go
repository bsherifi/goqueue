package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betim/goqueue/api"
	"github.com/betim/goqueue/queue"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	workers := flag.Int("workers", 4, "number of worker goroutines")
	flag.Parse()

	fmt.Printf("GoQueue starting (port=%d, workers=%d)\n", *port, *workers)

	// Create a context that cancels on SIGINT or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create the queue manager and worker pool
	manager := queue.NewManager(100)
	pool := queue.NewWorkerPool(*workers, manager.JobChan)
	pool.Start(ctx)

	// Start the HTTP server in a goroutine so it doesn't block
	server := api.NewServer(*port, manager)
	go func() {
		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			fmt.Printf("HTTP server error: %s\n", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("\nShutting down...")

	// Give the HTTP server 5 seconds to finish active requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("HTTP server shutdown error: %s\n", err)
	}

	// Close the job channel so workers stop receiving new jobs
	close(manager.JobChan)

	// Wait for workers to finish their current jobs
	pool.Stop()

	fmt.Println("GoQueue stopped")
}
