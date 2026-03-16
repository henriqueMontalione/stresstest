package runner

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Summary holds the aggregated results of a load test run.
type Summary struct {
	Total       int
	Duration    time.Duration
	StatusCodes map[int]int
}

// Run executes a load test against url with the given total requests and concurrency.
// It returns a Summary with timing and status code distribution.
func Run(ctx context.Context, url string, requests, concurrency int) Summary {
	jobs := make(chan struct{}, requests)
	for i := 0; i < requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make(chan int, requests)

	client := &http.Client{Timeout: 30 * time.Second}

	var completed atomic.Int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for range jobs {
				code := doRequest(ctx, client, url)
				results <- code
				completed.Add(1)
			}
		}()
	}

	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n := completed.Load()
				fmt.Printf("\rProgress: %d/%d requests completed", n, requests)
			case <-stopProgress:
				return
			}
		}
	}()

	wg.Wait()
	close(stopProgress)

	duration := time.Since(start)

	fmt.Printf("\rProgress: %d/%d requests completed\n", requests, requests)

	close(results)

	statusCodes := make(map[int]int)
	for code := range results {
		statusCodes[code]++
	}

	return Summary{
		Total:       requests,
		Duration:    duration,
		StatusCodes: statusCodes,
	}
}

func doRequest(ctx context.Context, client *http.Client, url string) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	return resp.StatusCode
}
