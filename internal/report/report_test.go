package report_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/henriqueMontalione/stresstest/internal/report"
	"github.com/henriqueMontalione/stresstest/internal/runner"
)

func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrint_ContainsExpectedSections(t *testing.T) {
	s := runner.Summary{
		Total:    100,
		Duration: 2500 * time.Millisecond,
		StatusCodes: map[int]int{
			200: 90,
			404: 10,
		},
	}

	out := captureStdout(func() { report.Print(s) })

	checks := []string{
		"Load Test Report",
		"Total time:",
		"Total requests:",
		"HTTP Status Distribution:",
		"200",
		"404",
		"90 requests",
		"10 requests",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q\nGot:\n%s", want, out)
		}
	}
}

func TestPrint_ConnectionErrorLabel(t *testing.T) {
	s := runner.Summary{
		Total:    5,
		Duration: time.Second,
		StatusCodes: map[int]int{
			0: 5,
		},
	}

	out := captureStdout(func() { report.Print(s) })

	if !strings.Contains(out, "Connection error") {
		t.Errorf("expected 'Connection error' label for code 0\nGot:\n%s", out)
	}
}

func TestPrint_StatusCodesAreSorted(t *testing.T) {
	s := runner.Summary{
		Total:    30,
		Duration: time.Second,
		StatusCodes: map[int]int{
			500: 5,
			200: 20,
			404: 5,
		},
	}

	out := captureStdout(func() { report.Print(s) })

	pos200 := strings.Index(out, "200")
	pos404 := strings.Index(out, "404")
	pos500 := strings.Index(out, "500")

	if !(pos200 < pos404 && pos404 < pos500) {
		t.Errorf("expected status codes in ascending order (200 < 404 < 500)\nGot:\n%s", out)
	}
}

func TestPrint_TotalRequestsMatchesSummary(t *testing.T) {
	s := runner.Summary{
		Total:    42,
		Duration: 500 * time.Millisecond,
		StatusCodes: map[int]int{
			200: 42,
		},
	}

	out := captureStdout(func() { report.Print(s) })

	if !strings.Contains(out, fmt.Sprintf("%d", s.Total)) {
		t.Errorf("expected total requests %d in output\nGot:\n%s", s.Total, out)
	}
}
