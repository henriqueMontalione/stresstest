package runner_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/henriqueMontalione/stresstest/internal/runner"
)

func TestRun_ExactRequestCount(t *testing.T) {
	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	summary := runner.Run(context.Background(), srv.URL, 50, 5)

	if summary.Total != 50 {
		t.Errorf("expected Total=50, got %d", summary.Total)
	}
	if count != 50 {
		t.Errorf("expected 50 HTTP hits, got %d", count)
	}
}

func TestRun_StatusCodeDistribution(t *testing.T) {
	codes := []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}
	idx := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(codes[idx%len(codes)])
		idx++
	}))
	defer srv.Close()

	summary := runner.Run(context.Background(), srv.URL, 9, 3)

	if len(summary.StatusCodes) == 0 {
		t.Fatal("expected status code distribution, got empty map")
	}
	total := 0
	for _, n := range summary.StatusCodes {
		total += n
	}
	if total != 9 {
		t.Errorf("expected status code counts to sum to 9, got %d", total)
	}
}

func TestRun_ConnectionError(t *testing.T) {
	summary := runner.Run(context.Background(), "http://127.0.0.1:1", 5, 5)

	if summary.StatusCodes[0] != 5 {
		t.Errorf("expected 5 connection errors (code 0), got %d", summary.StatusCodes[0])
	}
}

func TestRun_ConcurrencyHigherThanOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	summary := runner.Run(context.Background(), srv.URL, 100, 20)

	if summary.Total != 100 {
		t.Errorf("expected Total=100, got %d", summary.Total)
	}
	if summary.StatusCodes[http.StatusOK] != 100 {
		t.Errorf("expected 100 x 200 OK, got %d", summary.StatusCodes[http.StatusOK])
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summary := runner.Run(ctx, srv.URL, 10, 2)

	// all requests should fail due to cancelled context
	if summary.StatusCodes[0] != 10 {
		t.Errorf("expected 10 errors with cancelled context, got %d ok and %d errors",
			summary.StatusCodes[http.StatusOK], summary.StatusCodes[0])
	}
}
