# revproxy-go

A reverse proxy server written in Go. It matches client endpoints to one of multiple backend URLs, selected based on customizable load-balancing strategies.

## Features

- **Load Balancing Strategies**:
  - Round-robin
  - Weighted random
  - Pure random

- **Ban System**: Temporarily disables poorly performing backends based on:
  - Response status codes
  - Response body keyword matching

- **SOCKS5 Proxy Support**: Each backend can be optionally attached to a SOCKS5 tunnel.

- **HTTPS/TLS Support**: Configure certificates in the config file.



## Usage


Go **v1.22** or higher is required.

1. **Clone the repository**:

   ```bash
   git clone https://github.com/abswn/revproxy-go.git
   cd revproxy-go
    ```

2. **Run tests**:

   ```bash
   go test ./... -v
   ```

3. **Build the binary**:

   ```bash
   go build -o revproxy-go .
   ```

4. **Run the server**:

   ```bash
   ./revproxy-go
   ```

## Directory Structure
```plaintext
revproxy-go/
├── cmd/                   # Future command-line tools
├── configs/
│   ├── config.yaml        # Main server configuration
│   └── endpoints/         # Per-site endpoint configurations
├── internal/
│   ├── ban/
│   ├── cert/
│   ├── config/
│   ├── forward/
│   └── strategy/
├── main.go
└── README.md
```
## Configuration

### Global Configuration (`configs/config.yaml`)

```yaml
port: 44562

# Path to certs or leave empty to use without encryption 
https_cert_path: ""
https_key_path: ""

log:
  level: "info"             # Options: debug, info, warn, error, off
  output: "logs/output.log" # "stdout" or a file path like "logs/output.log"
  format: "text"            # "text" or "json"
```

### Endpoint Configuration (`configs/endpoints/example.yaml`)

Multiple YAML files can be used to separate the endpoints logically. All the configs with `enabled` flag `true` will be active.

```yaml
enabled: true

endpoints:
  "/api":
    strategy: round-robin # random, weighted, round-robin
    urls:
      - url: "https://example.com/api1"
        # Optional: socks5, username, password
        socks5: "127.0.0.1:1080"
        username: "user1"
        password: "pass1"
      - url: "https://example.com/api2"
      - url: "https://example.com/api3"
    ban:
      - match : ["429", "try after some time"]
        duration: 30 # temporarily disables a backend for 30 secs
      - match : ["500"]
        duration: 3600

  "/path":
    strategy: weighted
    urls:
      - url: "https://example.com/api1"
        weight: 50 # Required for weighted strategy
      - url: "https://example.com/api2"
        weight: 30
      - url: "https://example.com/api3"
        weight: 20

# Applies to all endpoints and is overridden locally
global_ban:
  - match: ["429"]
    duration: 30 # in seconds
  - match: ["503", "out of capacity"]
    duration: 3600
```

### Explanation

* `enabled`: Whether this config file is active
* `endpoints`: Map of path to backend strategy and URLs
* `strategy`: Can be either round-robin, weighted or pure random
* `urls`: List of backend definitions

  * `url`: Backend target URL
  * `socks5`: Optional SOCKS5 proxy address
  * `weight`: Used only with `weighted` strategy
* `ban` / `global_ban`: The `global_ban` rules apply to all endpoints in the config. The local `ban` rules add to it or override it. Multiple keywords can be written in the same line.



Start the server and send requests.

```bash
curl http://localhost:44562/api
```
This will use one of the three backend servers to fetch the result. The selection of the backend server is done in round-robin format. If no healthy backends are available and all have been temporarily disabled, the server responds with `503 Service Unavailable`.

## Logging

Configured via `config.yaml`:

* `level`: `debug`, `info`, `warn`, `error`, `off`
* `output`: `stdout` or path to log file
* `format`: `text` or `json`

## HTTPS/TLS Support

* If cert/key paths are provided, they are used.
* Otherwise fallback to non-encrypted HTTP.

## License

MIT License


