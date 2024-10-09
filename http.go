package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func timerHandler(t *SecondsTimer) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			log.Printf("GET request from %s", req.Header.Get("User-Agent"))
			// Respond with remaining and end time for timer
			fmt.Fprintf(res, "{\"seconds\":%.f,\"end\":\"%s\"}", t.TimeRemaining().Seconds(), t.end.Format(time.RFC3339))
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
			fmt.Fprintf(res, "{\"seconds\":%.f,\"end\":\"%s\"}", t.TimeRemaining().Seconds(), t.end.Format(time.RFC3339))
		default:
			http.Error(res, "Not supported", 400)
		}
	}
}
