package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// HTTP request/response limits and validation constants
const (
	minTimerSeconds     = 0          // Minimum timer value (0 = fire immediately)
	maxTimerSeconds     = 86400 * 30 // Maximum timer value: 30 days in seconds
	maxRequestBodyBytes = 1024       // Maximum request body size to prevent DoS attacks
)

// inputTimer represents the JSON payload for setting a timer via PUT request.
type inputTimer struct {
	Seconds int `json:"seconds"` // Number of seconds until timer fires
}

// outputTimer represents the JSON response for GET requests showing timer status.
type outputTimer struct {
	Seconds int    `json:"seconds"` // Seconds remaining until timer fires
	End     string `json:"end"`     // ISO8601 timestamp when timer will fire
}

// timerHandler creates an HTTP handler for managing the timer.
// It supports:
//   - GET: Returns current timer status (seconds remaining and end time)
//   - PUT: Sets a new timer duration (0 to 30 days)
//
// The handler is thread-safe and can handle concurrent requests.
func timerHandler(t *SecondsTimer) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		// GET: Return current timer status
		case http.MethodGet:
			log.Printf("GET request from %s", req.Header.Get("User-Agent"))

			// Build response with current timer state
			output := outputTimer{
				Seconds: int(math.Round(t.TimeRemaining().Seconds())),
				End:     t.End().Format(time.RFC3339),
			}
			jsonData, err := json.Marshal(output)
			if err != nil {
				// This should never happen with our simple struct, but handle it anyway
				log.Printf("GET request failed with: %s", err)
				http.Error(res, "Unable to output timer", http.StatusInternalServerError)
			} else {
				res.Header().Set("Content-Type", "application/json")
				log.Printf("GET response: %s", string(jsonData))
				res.Write(jsonData)
			}

		// PUT: Set a new timer duration
		case http.MethodPut:
			log.Printf("PUT request from %s", req.Header.Get("User-Agent"))

			// Limit request body size to prevent DoS attacks
			req.Body = http.MaxBytesReader(res, req.Body, maxRequestBodyBytes)

			// Parse and validate JSON input
			var jsonData inputTimer
			decoder := json.NewDecoder(req.Body)
			decoder.DisallowUnknownFields() // Reject unknown fields for strict validation
			err := decoder.Decode(&jsonData)
			if err != nil {
				// Log detailed error but return generic message to client for security
				log.Printf("PUT request decode error: %s", err)
				http.Error(res, "Invalid request format", http.StatusBadRequest)
				return
			}

			// Validate timer bounds
			if jsonData.Seconds < minTimerSeconds {
				log.Printf("PUT request failed: timer value too small (%d)", jsonData.Seconds)
				http.Error(res, "Timer must be positive", http.StatusBadRequest)
				return
			}
			if jsonData.Seconds > maxTimerSeconds {
				log.Printf("PUT request failed: timer value too large (%d)", jsonData.Seconds)
				http.Error(res, fmt.Sprintf("Timer exceeds maximum duration (%d seconds)", maxTimerSeconds), http.StatusBadRequest)
				return
			}

			// All validation passed - set the timer
			t.Reset(time.Duration(jsonData.Seconds) * time.Second)
			log.Printf("Set timer to %d seconds", jsonData.Seconds)

			// Return success response
			res.Header().Set("Content-Type", "application/json")
			res.Write([]byte(`{"success":true}`))

		default:
			// Reject unsupported HTTP methods
			log.Printf("HTTP request not supported")
			http.Error(res, "Not supported", http.StatusNotImplemented)
		}
	}
}
