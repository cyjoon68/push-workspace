package domain

import (
	"testing"
	"time"
)

func mustTime(t *testing.T) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, "2025-01-02T03:04:05.000000006Z")
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
