package report

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/henriqueMontalione/stresstest/internal/runner"
)

// Print formats and prints the load test summary to stdout.
func Print(s runner.Summary) {
	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Println("  Load Test Report")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("  Total time:       %.3fs\n", s.Duration.Seconds())
	fmt.Printf("  Total requests:   %d\n", s.Total)
	fmt.Println()
	fmt.Println("  HTTP Status Distribution:")

	codes := make([]int, 0, len(s.StatusCodes))
	for code := range s.StatusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	for _, code := range codes {
		count := s.StatusCodes[code]
		label := statusLabel(code)
		fmt.Printf("    %-3d  %-20s %d requests\n", code, label, count)
	}

	fmt.Println("------------------------------------------------------------")
}

func statusLabel(code int) string {
	if code == 0 {
		return "Connection error"
	}
	text := http.StatusText(code)
	if text == "" {
		return "Unknown"
	}
	return text
}
