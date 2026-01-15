package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTimerHandlerGET tests GET requests to /timer
func TestTimerHandlerGET(t *testing.T) {
	timer := NewSecondsTimer(10 * time.Second)
	defer timer.Stop()

	handler := timerHandler(timer)
	req := httptest.NewRequest(http.MethodGet, "/timer", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("GET returned status %d, expected %d", rec.Code, http.StatusOK)
	}

	// Check Content-Type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, expected application/json", contentType)
	}

	// Parse response
	var response outputTimer
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify response has expected fields
	if response.Seconds < 9 || response.Seconds > 10 {
		t.Errorf("Response seconds = %d, expected ~10", response.Seconds)
	}

	if response.End == "" {
		t.Error("Response end time is empty")
	}

	// Verify end time is valid RFC3339
	if _, err := time.Parse(time.RFC3339, response.End); err != nil {
		t.Errorf("End time not valid RFC3339: %v", err)
	}
}

// TestTimerHandlerPUTValid tests valid PUT requests
func TestTimerHandlerPUTValid(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)

	testCases := []struct {
		name    string
		seconds int
	}{
		{"Zero seconds", 0},
		{"One second", 1},
		{"One minute", 60},
		{"One hour", 3600},
		{"One day", 86400},
		{"Maximum (30 days)", maxTimerSeconds},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]int{"seconds": tc.seconds}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPut, "/timer", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("PUT returned status %d, expected %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			// Check response
			var response map[string]bool
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			if !response["success"] {
				t.Error("Expected success:true in response")
			}

			// Verify timer was actually set
			remaining := timer.TimeRemaining()
			expectedRemaining := time.Duration(tc.seconds) * time.Second
			diff := (remaining - expectedRemaining).Abs()

			if diff > time.Second {
				t.Errorf("Timer not set correctly: remaining=%v, expected=%v", remaining, expectedRemaining)
			}
		})
	}
}

// TestTimerHandlerPUTInvalid tests invalid PUT requests
func TestTimerHandlerPUTInvalid(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)

	testCases := []struct {
		name           string
		payload        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Negative seconds",
			payload:        `{"seconds":-1}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Timer must be positive",
		},
		{
			name:           "Too large",
			payload:        `{"seconds":99999999}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Timer exceeds maximum duration",
		},
		{
			name:           "Invalid JSON",
			payload:        `{"seconds":`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
		{
			name:           "Wrong type",
			payload:        `{"seconds":"not a number"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
		{
			name:           "Unknown field",
			payload:        `{"seconds":10,"extra":"field"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
		{
			name:           "Empty body",
			payload:        ``,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
		{
			name:           "Missing seconds field",
			payload:        `{}`,
			expectedStatus: http.StatusOK, // seconds defaults to 0, which is valid
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/timer", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Status = %d, expected %d", rec.Code, tc.expectedStatus)
			}

			if tc.expectedError != "" {
				body := rec.Body.String()
				if !strings.Contains(body, tc.expectedError) {
					t.Errorf("Error message = %q, expected to contain %q", body, tc.expectedError)
				}
			}
		})
	}
}

// TestTimerHandlerPUTOversizedBody tests request body size limit
func TestTimerHandlerPUTOversizedBody(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)

	// Create payload larger than maxRequestBodyBytes (1024)
	largePayload := `{"seconds":60,"padding":"` + strings.Repeat("x", 2000) + `"}`

	req := httptest.NewRequest(http.MethodPut, "/timer", strings.NewReader(largePayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Oversized request status = %d, expected %d", rec.Code, http.StatusBadRequest)
	}
}

// TestTimerHandlerUnsupportedMethod tests unsupported HTTP methods
func TestTimerHandlerUnsupportedMethod(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch, http.MethodHead}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/timer", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != http.StatusNotImplemented {
				t.Errorf("%s returned status %d, expected %d", method, rec.Code, http.StatusNotImplemented)
			}

			if !strings.Contains(rec.Body.String(), "Not supported") {
				t.Errorf("Error message doesn't contain 'Not supported'")
			}
		})
	}
}

// TestTimerHandlerConcurrent tests concurrent requests
func TestTimerHandlerConcurrent(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)

	// Launch multiple concurrent requests
	const concurrency = 50
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			// Alternate between GET and PUT
			if id%2 == 0 {
				req := httptest.NewRequest(http.MethodGet, "/timer", nil)
				rec := httptest.NewRecorder()
				handler(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("Concurrent GET %d failed: status=%d", id, rec.Code)
				}
			} else {
				payload := `{"seconds":10}`
				req := httptest.NewRequest(http.MethodPut, "/timer", strings.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				handler(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("Concurrent PUT %d failed: status=%d", id, rec.Code)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// TestInputTimerJSON tests inputTimer JSON marshaling
func TestInputTimerJSON(t *testing.T) {
	input := inputTimer{Seconds: 123}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal inputTimer: %v", err)
	}

	expected := `{"seconds":123}`
	if string(data) != expected {
		t.Errorf("JSON = %s, expected %s", string(data), expected)
	}

	// Test unmarshaling
	var decoded inputTimer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Seconds != input.Seconds {
		t.Errorf("Decoded seconds = %d, expected %d", decoded.Seconds, input.Seconds)
	}
}

// TestOutputTimerJSON tests outputTimer JSON marshaling
func TestOutputTimerJSON(t *testing.T) {
	output := outputTimer{
		Seconds: 456,
		End:     "2026-01-15T12:00:00Z",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal outputTimer: %v", err)
	}

	// Verify it contains expected fields
	str := string(data)
	if !strings.Contains(str, `"seconds":456`) {
		t.Errorf("JSON missing seconds field: %s", str)
	}
	if !strings.Contains(str, `"end":"2026-01-15T12:00:00Z"`) {
		t.Errorf("JSON missing end field: %s", str)
	}
}

// BenchmarkTimerHandlerGET benchmarks GET requests
func BenchmarkTimerHandlerGET(b *testing.B) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)
	req := httptest.NewRequest(http.MethodGet, "/timer", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

// BenchmarkTimerHandlerPUT benchmarks PUT requests
func BenchmarkTimerHandlerPUT(b *testing.B) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	handler := timerHandler(timer)
	payload := `{"seconds":60}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPut, "/timer", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}
