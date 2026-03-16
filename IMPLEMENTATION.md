# Implementation Notes

Detailed explanation of each file, the decisions made, and the reasoning behind them.

---

## internal/runner/runner.go

Core of the application. Responsible for distributing requests across concurrent workers and collecting results.

**Jobs channel — exact request count guarantee**

The jobs channel is buffered with capacity equal to `requests`, filled with N empty structs, and closed before any worker starts. Workers range over the channel — when it drains, they exit naturally. This eliminates any need for an atomic counter to control how many requests were made.

```go
jobs := make(chan struct{}, requests)
for i := 0; i < requests; i++ {
    jobs <- struct{}{}
}
close(jobs)
```

**Results channel — thread-safe collection without mutex**

Each worker sends its HTTP status code to a buffered results channel. A single drain loop after all workers finish reads every value and builds the aggregation map. No shared state, no mutex needed.

```go
results := make(chan int, requests)
// workers send to results
// after wg.Wait(), drain:
for code := range results {
    statusCodes[code]++
}
```

**Progress bar — atomic counter + ticker**

`atomic.Int64` is incremented by each worker after every completed request. A dedicated goroutine reads it every 100ms and overwrites the current terminal line with `\r`. The goroutine is stopped via a `stopProgress` channel after `wg.Wait()` returns, then the final count is printed with `\n`.

```go
var completed atomic.Int64
// workers: completed.Add(1) after each request
// progress goroutine: fmt.Printf("\rProgress: %d/%d ...", completed.Load(), requests)
```

**HTTP client with explicit timeout**

The default `http.DefaultClient` has no timeout. Under load, slow servers can cause goroutines to hang indefinitely. A dedicated client with a 30-second timeout is created once and shared across all workers (it is safe for concurrent use).

```go
client := &http.Client{Timeout: 30 * time.Second}
```

**Context propagation**

Every HTTP request is created with `http.NewRequestWithContext(ctx, ...)`. The context comes from `main.go` and can carry cancellation or deadline signals. Connection errors return status code `0` to distinguish them from valid HTTP responses.

---

## internal/report/report.go

Responsible only for formatting and printing. No concurrency, no HTTP knowledge.

**Sorted status codes**

The `StatusCodes` map is iterated in sorted order so the report is deterministic and easy to read regardless of which codes arrived first.

**Status labels**

`http.StatusText(code)` provides the standard label for every HTTP code. Code `0` is treated as "Connection error" since it represents failed requests that never reached the server.

---

## cmd/stresstest/main.go

Entry point. Parses flags, validates input, wires dependencies, runs the test.

**Validation rules**

- `--url` must be non-empty
- `--requests` must be >= 1
- `--concurrency` must be >= 1 and <= `--requests`

On any validation failure, the error is printed to `stderr`, usage is shown, and the process exits with code 1.

**Wiring**

`main` is the only place that knows about both `runner` and `report`. It calls `runner.Run()`, receives a `Summary`, and passes it to `report.Print()`. The two packages never import each other.

---

## Dockerfile

Multi-stage build to produce a minimal final image.

**Stage 1 — builder (`golang:1.26-alpine`)**

- `CGO_ENABLED=0`: produces a fully static binary with no libc dependency
- `GOOS=linux`: cross-compiles for Linux even when building on macOS
- `-ldflags="-s -w"`: strips debug symbols and DWARF info, reducing binary size

**Stage 2 — runner (`scratch`)**

- `scratch` is an empty image — no shell, no OS, just the binary
- `ca-certificates.crt` is copied from the builder stage so HTTPS requests work correctly inside the container
- `ENTRYPOINT` (not `CMD`) means flags passed to `docker run` go directly to the binary:

```
docker run <image> --url=http://... --requests=1000 --concurrency=10
```

Final image size: ~5 MB.
