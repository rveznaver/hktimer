package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

const (
	minTimerSeconds = 0
	maxTimerSeconds = 86400 * 30 // 30 days maximum
)

type inputTimer struct {
	Seconds int `json:"seconds"`
}

type outputTimer struct {
	Seconds int    `json:"seconds"`
	End     string `json:"end"`
}

func timerHandler(t *SecondsTimer) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		// GET requests
		case http.MethodGet:
			log.Printf("GET request from %s", req.Header.Get("User-Agent"))
			// Respond with remaining seconds and end time for timer
			output := outputTimer{Seconds: int(math.Round(t.TimeRemaining().Seconds())), End: t.End().Format(time.RFC3339)}
			jsonData, err := json.Marshal(output)
			if err != nil {
				log.Printf("GET request failed with: %s", err)
				http.Error(res, "Unable to output timer", http.StatusInternalServerError)
			} else {
				log.Printf("GET response: %s", string(jsonData))
				fmt.Fprintf(res, string(jsonData))
			}
		// PUT requests
		case http.MethodPut:
			log.Printf("PUT request from %s", req.Header.Get("User-Agent"))
			// Decode JSON data
			var jsonData inputTimer
			decoder := json.NewDecoder(req.Body)
			decoder.DisallowUnknownFields()
			err := decoder.Decode(&jsonData)
			if err != nil {
				log.Printf("PUT request failed with: %s", err)
				http.Error(res, err.Error(), http.StatusBadRequest)
			} else {
				if jsonData.Seconds < minTimerSeconds {
					log.Printf("PUT request failed: timer value too small (%d)", jsonData.Seconds)
					http.Error(res, "Timer must be positive", http.StatusBadRequest)
				} else if jsonData.Seconds > maxTimerSeconds {
					log.Printf("PUT request failed: timer value too large (%d)", jsonData.Seconds)
					http.Error(res, fmt.Sprintf("Timer exceeds maximum duration (%d seconds)", maxTimerSeconds), http.StatusBadRequest)
				} else {
					// Set timer
					t.Reset(time.Duration(jsonData.Seconds) * time.Second)
					log.Printf("Set timer to %d seconds", jsonData.Seconds)
				}
			}
		default:
			log.Printf("HTTP request not supported")
			http.Error(res, "Not supported", http.StatusNotImplemented)
		}
	}
}
