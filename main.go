// Package main implements a HomeKit-compatible timer switch that can be controlled
// via both HomeKit and an HTTP API. When the timer expires, it automatically
// turns on a virtual HomeKit switch.
//
// The HTTP API supports:
//   - GET /timer: Check timer status
//   - PUT /timer: Set timer duration (0 to 30 days)
//
// The implementation is thread-safe and supports graceful shutdown.
package main

import (
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"

	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Command-line flags
var (
	port     = flag.Int("port", 30001, "HTTP server port (1-65535)")
	useNvram = flag.Bool("nvram", false, "Use NVRAM storage instead of filesystem (for FreshTomato routers)")
)

func main() {
	flag.Parse()

	// Validate port range to prevent invalid configurations
	if *port < 1 || *port > 65535 {
		log.Fatalf("Port must be between 1 and 65535, got: %d", *port)
	}

	// Create a HomeKit switch accessory that can be controlled via HomeKit apps
	a := accessory.NewSwitch(accessory.Info{
		Name: "timer",
	})

	// Log when the switch is controlled through HomeKit
	a.Switch.On.OnValueRemoteUpdate(func(on bool) {
		if on {
			log.Println("Switching on remotely")
		} else {
			log.Println("Switching off remotely")
		}
	})

	// Select storage backend based on command-line flag
	var store hap.Store
	if *useNvram {
		log.Println("Using NVRAM storage")
		store = NewNvramStore()
	} else {
		log.Println("Using filesystem storage (./db)")
		store = hap.NewFsStore("./db")
	}

	// Create the HomeKit Accessory Protocol (HAP) server
	s, err := hap.NewServer(store, a.A)
	if err != nil {
		log.Fatal("Failed to create HAP server: ", err)
	}

	// Configure the HTTP server address
	s.Addr = fmt.Sprintf(":%d", *port)

	// Create a timer in stopped state (won't fire until set via HTTP API)
	// We use time.Hour as a placeholder duration since we stop it immediately
	t := NewSecondsTimer(time.Hour)
	t.Stop()

	// Create a context for coordinating graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Start a goroutine to wait for timer expiration and trigger the HomeKit switch
	// This goroutine will run until shutdown and is context-aware for clean exit
	go func() {
		for {
			select {
			case <-t.C():
				// Timer expired - turn on the HomeKit switch
				log.Println("Switching on via timer")
				a.Switch.On.SetValue(true)
			case <-ctx.Done():
				// Shutdown requested - clean up timer and exit goroutine
				t.Stop()
				log.Println("Timer goroutine stopped")
				return
			}
		}
	}()

	// Register the HTTP handler for the /timer endpoint
	s.ServeMux().HandleFunc("/timer", timerHandler(t))

	// Setup signal handling for graceful shutdown
	// Buffered channel ensures we don't miss signals during processing
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // Handle Ctrl+C and kill signals

	// Start a goroutine to handle shutdown signals
	go func() {
		<-c // Block until we receive a signal
		log.Println("Stopping hktimer")

		// Stop receiving more signals
		signal.Stop(c)

		// Cancel the context, triggering graceful shutdown of:
		// - The HAP server
		// - The timer goroutine
		// - Any other context-aware components
		cancel()
	}()

	// Start the server (blocks until context is cancelled)
	log.Println("Starting hktimer")
	s.ListenAndServe(ctx)
}
