# stresstest

A fast, concurrent HTTP load testing CLI written in Go. Distribute N requests across C concurrent workers and get a detailed execution report — no external dependencies, single binary, Docker-ready.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Execution Flow](#execution-flow)
- [Parameters](#parameters)
- [Running Locally](#running-locally)
- [Running with Docker](#running-with-docker)
- [Report Example](#report-example)
- [Building the Docker Image](#building-the-docker-image)

---

## Overview

`stresstest` is a command-line tool for load testing HTTP services. You specify a target URL, a total number of requests, and a concurrency level — the tool distributes the work, executes all requests, and prints a summary report with timing and HTTP status code distribution.

```
docker run henriquem/stresstest --url=http://example.com --requests=1000 --concurrency=10
```

---

## Features

- Exact request count guarantee — no under or over-shooting
- Configurable concurrency via worker pool
- Live progress indicator during execution
- Detailed report: total time, status code distribution
- Zero external dependencies — pure Go stdlib
- Minimal Docker image (~5 MB) via multi-stage build with `scratch`

---

## Architecture

```
stresstest/
├── cmd/
│   └── stresstest/
│       └── main.go          # Entry point: flag parsing and wiring
├── internal/
│   ├── runner/
│   │   └── runner.go        # Concurrency engine: worker pool, job distribution
│   └── report/
│       └── report.go        # Result aggregation and console report
├── Dockerfile
├── go.mod
└── README.md
```

### Layer responsibilities

| Layer | Responsibility |
|---|---|
| `cmd/stresstest` | Parse CLI flags, validate input, call runner, print report |
| `internal/runner` | Distribute requests across workers, collect results, measure time |
| `internal/report` | Aggregate status codes, format and print the final report |

### Design decisions

| Concern | Approach | Reason |
|---|---|---|
| Flag parsing | `flag` (stdlib) | No external deps needed |
| HTTP client | `net/http` (stdlib) | Full control, configurable timeout |
| Concurrency | Goroutines + buffered channels | Idiomatic Go, no libs |
| Result collection | Dedicated result channel | Avoids shared state and mutexes |
| Progress feedback | In-place `\r` counter | Zero deps, immediate UX value |
| Docker image | Multi-stage (`golang` → `scratch`) | Minimal final image size |

---

## Execution Flow

```
main
  └─ runner.Run(url, requests, concurrency)
        │
        ├─ fill jobs channel with N tickets (buffered, capacity = requests)
        │
        ├─ start C worker goroutines
        │     └─ each worker:
        │           loop until jobs channel is empty
        │             → consume one ticket
        │             → HTTP GET to target URL
        │             → send Result{statusCode, err} to results channel
        │
        ├─ progress goroutine: reads completed count → prints live counter via \r
        │
        ├─ wait for all workers to finish (WaitGroup)
        │
        └─ return Summary{duration, map[statusCode]count}

report.Print(summary)
  └─ print total time, total requests, HTTP 200 count, other codes
```

**Why channels instead of a shared slice + mutex?**

Each worker communicates results through a dedicated channel. A single collector goroutine reads from it and builds the aggregation map. This eliminates shared mutable state entirely — no mutex, no race condition possible.

**Why a buffered jobs channel?**

Filling a buffered channel of size N with N tickets before starting workers means:
- Workers start immediately with no coordination overhead
- The channel itself enforces the exact request count — when it's empty, workers exit naturally
- No atomic counter or additional synchronization needed

---

## Parameters

| Flag | Type | Required | Description |
|---|---|---|---|
| `--url` | string | yes | Target URL to test |
| `--requests` | int | yes | Total number of requests to perform |
| `--concurrency` | int | yes | Number of simultaneous workers |

### Validation rules

- `--url` must be non-empty
- `--requests` must be ≥ 1
- `--concurrency` must be ≥ 1 and ≤ `--requests`

---

## Running Locally

**Prerequisites:** Go 1.26.1+

```bash
# Clone the repository
git clone https://github.com/<your-user>/stresstest.git
cd stresstest

# Run directly
go run ./cmd/stresstest --url=http://google.com --requests=100 --concurrency=10

# Or build and run the binary
go build -o stresstest ./cmd/stresstest
./stresstest --url=http://google.com --requests=100 --concurrency=10
```

---

## Running with Docker

### Pull and run (when published)

```bash
docker run henriquem/stresstest --url=http://google.com --requests=1000 --concurrency=10
```

### Build and run locally

```bash
# Build the image
docker build -t stresstest .

# Run the load test
docker run stresstest --url=http://google.com --requests=1000 --concurrency=10
```

### Testing against a local service

When the target service runs on your host machine, use `host.docker.internal` instead of `localhost`:

```bash
docker run stresstest --url=http://host.docker.internal:8080 --requests=500 --concurrency=20
```

---

## Report Example

```
Starting load test...
Target:      http://google.com
Requests:    1000
Concurrency: 10

Progress: 1000/1000 requests completed

------------------------------------------------------------
  Load Test Report
------------------------------------------------------------
  Total time:       4.832s
  Total requests:   1000

  HTTP Status Distribution:
    200   OK              850 requests
    301   Moved           120 requests
    429   Too Many Req     25 requests
    500   Internal Err      5 requests
------------------------------------------------------------
```

---

## Building the Docker Image

The Dockerfile uses a **multi-stage build**:

1. **Stage `builder`** — compiles the binary using `golang:1.26-alpine`
2. **Stage `runner`** — copies only the binary into a `scratch` (empty) image

The result is an image of ~5 MB containing only the static binary and TLS certificates.

```bash
docker build -t stresstest .
```

To tag for Docker Hub:

```bash
docker build -t <your-dockerhub-user>/stresstest:latest .
docker push <your-dockerhub-user>/stresstest:latest
```
