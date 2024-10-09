package main

import (
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"

	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create the switch accessory
	a := accessory.NewSwitch(accessory.Info{
		Name: "timer",
	})

	// Log control through HomeKit
	a.Switch.On.OnValueRemoteUpdate(func(on bool) {
		if on == true {
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
		// stop if an error happens
		log.Panic(err)
	}

	// TODO: Make variable from cmdline
	s.Addr = ":30001"

	// Create a timer for future use
	t := NewSecondsTimer(0)
	if !t.timer.Stop() {
		<-t.timer.C
	}

	// Use a goroutine to wait for the timer to expire
	go func() {
		for {
			<-t.timer.C
			log.Println("Switching on via timer")
			a.Switch.On.SetValue(true)
			log.Println(a.Switch.On.Value())
		}
	}()

	s.ServeMux().HandleFunc("/timer", timerHandler(t))

	// Setup a listener for interrupts and SIGTERM signals to stop the server.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
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
