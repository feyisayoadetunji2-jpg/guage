package sep1

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type sleepRecorder struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (s *sleepRecorder) sleep(_ context.Context, delay time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waits = append(s.waits, delay)
	return nil
}

func (s *sleepRecorder) durations() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.waits...)
}

func clientForRetry(t *testing.T, server *httptest.Server) (*Client, *sleepRecorder) {
	t.Helper()
	c, _, _ := clientFor(t, server, map[string][]string{"example.com": {"93.184.216.34"}})
	c.opts.MaxRetries = defaultMaxRetries
	c.opts.Backoff = defaultBackoff
	sleeper := &sleepRecorder{}
	c.sleep = sleeper.sleep
	return c, sleeper
}

func TestRetryAfterBudget(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   []time.Duration
	}{
		{name: "one second", header: "1", want: []time.Duration{time.Second, time.Second}},
		{name: "no header", header: "", want: []time.Duration{defaultBackoff, 2 * defaultBackoff}},
		{name: "hostile header", header: "120", want: []time.Duration{maxRetryAfter, maxRetryAfter}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				if tc.header != "" {
					w.Header().Set("Retry-After", tc.header)
				}
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer server.Close()

			c, sleeper := clientForRetry(t, server)
			got := c.Fetch(context.Background(), "example.com")
			waits := sleeper.durations()
			t.Logf("%s: attempts=%d delays=%v result=%s", tc.name, requests.Load(), waits, got.FailedAt)

			if got.FailedAt != StepStatus {
				t.Fatalf("FailedAt = %v (%q), want status", got.FailedAt, got.Detail)
			}
			if got.Status != http.StatusTooManyRequests {
				t.Fatalf("Status = %d, want %d", got.Status, http.StatusTooManyRequests)
			}
			if want := int32(defaultMaxRetries + 1); requests.Load() != want {
				t.Fatalf("attempts = %d, want %d", requests.Load(), want)
			}
			if !reflect.DeepEqual(waits, tc.want) {
				t.Fatalf("delays = %v, want %v", waits, tc.want)
			}
		})
	}
}

func TestRetryBudgetRecoversOnThirdAttempt(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt := requests.Add(1)
		if attempt < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, tomlBody(mainnetPass))
	}))
	defer server.Close()

	c, sleeper := clientForRetry(t, server)
	got := c.Fetch(context.Background(), "example.com")
	waits := sleeper.durations()
	t.Logf("recovery: attempts=%d delays=%v result=%s", requests.Load(), waits, got.FailedAt)

	if got.FailedAt != StepOK {
		t.Fatalf("FailedAt = %v (%q), want ok", got.FailedAt, got.Detail)
	}
	if requests.Load() != 3 {
		t.Fatalf("attempts = %d, want 3", requests.Load())
	}
	want := []time.Duration{time.Second, time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("delays = %v, want %v", waits, want)
	}
}

func TestRetryBudgetStopsOnTerminalStatuses(t *testing.T) {
	cases := []struct {
		name   string
		status int
		step   Step
	}{
		{name: "not found", status: http.StatusNotFound, step: StepStatus},
		{name: "no content", status: http.StatusNoContent, step: StepNoKeys},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
			}))
			defer server.Close()

			c, sleeper := clientForRetry(t, server)
			got := c.Fetch(context.Background(), "example.com")
			waits := sleeper.durations()
			t.Logf("%s: attempts=%d delays=%v result=%s", tc.name, requests.Load(), waits, got.FailedAt)

			if got.FailedAt != tc.step {
				t.Fatalf("FailedAt = %v (%q), want %v", got.FailedAt, got.Detail, tc.step)
			}
			if requests.Load() != 1 {
				t.Fatalf("attempts = %d, want 1", requests.Load())
			}
			if len(waits) != 0 {
				t.Fatalf("delays = %v, want none", waits)
			}
		})
	}
}
