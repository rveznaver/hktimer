package main

import (
	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"

	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	// Create the switch accessory
	a := accessory.NewSwitch(accessory.Info{
		Name: "timer",
	})

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
		}
	}()

	s.ServeMux().HandleFunc("/timer", func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			log.Printf("GET request from %s", req.Header.Get("User-Agent"))
			// Respond with remaining and end time for timer
			fmt.Fprintf(res, "seconds=%.f\nend=%s", t.TimeRemaining().Seconds(), t.end.Format(time.RFC3339))
		case http.MethodPut:
			log.Printf("PUT request from %s", req.Header.Get("User-Agent"))
			// Parse form data
			if err := req.ParseForm(); err != nil {
				http.Error(res, "Unable to parse form", http.StatusBadRequest)
				return
			}
			// Retrieve the value from the form
			value := req.FormValue("seconds")
			if value == "" {
				http.Error(res, "Seconds not provided", http.StatusBadRequest)
				return
			}
			// Convert to string
			seconds, err := strconv.Atoi(value)
			if err != nil {
				http.Error(res, "Unable to read integer", http.StatusBadRequest)
				return
			}
			if seconds < 0 {
				http.Error(res, "Time has to be in the future", http.StatusBadRequest)
				return
			}
			// Set timer
			t.Reset(time.Duration(seconds) * time.Second)
			log.Printf("Set timer to %d seconds", seconds)
			// Respond with remaining and end time for timer
			fmt.Fprintf(res, "seconds=%.f\nend=%s", t.TimeRemaining().Seconds(), t.end.Format(time.RFC3339))
		default:
			http.Error(res, "Not supported", 400)
		}
	})

	// Setup a listener for interrupts and SIGTERM signals
	// to stop the server.
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
