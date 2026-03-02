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
	"github.com/betim/goqueue/store"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	workers := flag.Int("workers", 4, "number of worker goroutines")
	dbPath := flag.String("db", "", "path to SQLite database file (empty = in-memory)")
	flag.Parse()

	// Create the store — SQLite if --db is given, otherwise in-memory.
	var jobStore queue.Store
	if *dbPath != "" {
		s, err := store.NewSQLiteStore(*dbPath)
		if err != nil {
			fmt.Printf("Failed to open database: %s\n", err)
			os.Exit(1)
		}
		defer s.Close()
		jobStore = s
		fmt.Printf("GoQueue starting (port=%d, workers=%d, db=%s)\n", *port, *workers, *dbPath)
	} else {
		jobStore = store.NewMemoryStore()
		fmt.Printf("GoQueue starting (port=%d, workers=%d, db=in-memory)\n", *port, *workers)
	}

	// Create a context that cancels on SIGINT or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create the queue manager and worker pool
	manager := queue.NewManager(100, jobStore)
	pool := queue.NewWorkerPool(*workers, manager.JobChan, jobStore)
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
