# hktimer

Open-Source HomeKit Timer Switch

A simple HomeKit-compatible timer that can be controlled via both HomeKit and an HTTP API. When the timer expires, it automatically turns on a virtual HomeKit switch.

## Usage

```bash
# Build and run
go build
./hktimer

# Run with custom port
./hktimer -port 8080
```

## HTTP API

**Get timer status:**
```bash
curl http://localhost:30001/timer
```

**Set timer (seconds):**
```bash
curl -X PUT http://localhost:30001/timer \
  -H "Content-Type: application/json" \
  -d '{"seconds": 300}'
```

Timer range: 0 to 2,592,000 seconds (30 days)

## Testing

```bash
go test ./...
```

## Requirements

- Go 1.24+
- macOS or Linux
