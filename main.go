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

var (
	port = flag.Int("port", 30001, "HTTP server port")
)

func main() {
	flag.Parse()
	// Create the switch accessory
	a := accessory.NewSwitch(accessory.Info{
		Name: "timer",
	})

	// Log control through HomeKit
	a.Switch.On.OnValueRemoteUpdate(func(on bool) {
		if on {
			log.Println("Switching on remotely")
		} else {
			log.Println("Switching off remotely")
		}
	})

	// Store the data in the "./db" directory
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	s, err := hap.NewServer(fs, a.A)
	if err != nil {
		log.Fatal("Failed to create HAP server: ", err)
	}

	s.Addr = fmt.Sprintf(":%d", *port)

	// Create a stopped timer for future use
	t := NewSecondsTimer(time.Hour)
	t.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	// Use a goroutine to wait for the timer to expire
	go func() {
		for {
			select {
			case <-t.C():
				log.Println("Switching on via timer")
				a.Switch.On.SetValue(true)
			case <-ctx.Done():
				log.Println("Timer goroutine stopped")
				return
			}
		}
	}()

	s.ServeMux().HandleFunc("/timer", timerHandler(t))

	// Setup a listener for interrupts and SIGTERM signals to stop the server.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Stopping hktimer")
		// Stop delivering signals
		signal.Stop(c)
		// Cancel the context to stop the server
		cancel()
	}()

	// Run the server
	log.Println("Starting hktimer")
	s.ListenAndServe(ctx)
}
