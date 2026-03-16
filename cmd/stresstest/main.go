package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/henriqueMontalione/stresstest/internal/report"
	"github.com/henriqueMontalione/stresstest/internal/runner"
)

func main() {
	url := flag.String("url", "", "Target URL to test (required)")
	requests := flag.Int("requests", 0, "Total number of requests to perform (required)")
	concurrency := flag.Int("concurrency", 0, "Number of simultaneous workers (required)")

	flag.Parse()

	if err := validate(*url, *requests, *concurrency); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Starting load test...")
	fmt.Printf("Target:      %s\n", *url)
	fmt.Printf("Requests:    %d\n", *requests)
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Println()

	ctx := context.Background()
	summary := runner.Run(ctx, *url, *requests, *concurrency)

	report.Print(summary)
}

func validate(url string, requests, concurrency int) error {
	if url == "" {
		return fmt.Errorf("--url is required")
	}
	if requests < 1 {
		return fmt.Errorf("--requests must be >= 1")
	}
	if concurrency < 1 {
		return fmt.Errorf("--concurrency must be >= 1")
	}
	if concurrency > requests {
		return fmt.Errorf("--concurrency (%d) cannot exceed --requests (%d)", concurrency, requests)
	}
	return nil
}
