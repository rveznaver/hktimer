# hktimer

HomeKit timer switch with HTTP API. When the timer expires, it turns on a virtual HomeKit switch.

## Usage

```bash
go build
./hktimer [-port 30001] [-nvram]
```

Flags:
- `-port`: HTTP server port (default: 30001)
- `-nvram`: Use NVRAM storage instead of filesystem (for FreshTomato routers)

## HTTP API

```bash
# Get timer status
curl http://localhost:30001/timer

# Set timer (0 to 2592000 seconds)
curl -X PUT http://localhost:30001/timer -d '{"seconds": 300}'
```
