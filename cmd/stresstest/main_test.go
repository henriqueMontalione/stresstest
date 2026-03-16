package main

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		requests    int
		concurrency int
		wantErr     bool
	}{
		{"valid input", "http://example.com", 100, 10, false},
		{"concurrency equals requests", "http://example.com", 5, 5, false},
		{"missing url", "", 100, 10, true},
		{"requests zero", "http://example.com", 0, 10, true},
		{"requests negative", "http://example.com", -1, 10, true},
		{"concurrency zero", "http://example.com", 100, 0, true},
		{"concurrency negative", "http://example.com", 100, -1, true},
		{"concurrency exceeds requests", "http://example.com", 5, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.url, tt.requests, tt.concurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
