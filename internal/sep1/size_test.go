package sep1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

const sizeSentinel = "G151-OVERSIZE-DO-NOT-ECHO-"

func oversizedBody(size int) string {
	body := strings.Repeat(sizeSentinel, size/len(sizeSentinel)+1)
	return body[:size]
}

func TestOversizedBodyBoundary(t *testing.T) {
	cases := []struct {
		name      string
		size      int
		wantStep  Step
		wantBytes int
	}{
		{name: "exactly the cap", size: defaultMaxBody, wantStep: StepParse, wantBytes: defaultMaxBody},
		{name: "one over the cap", size: defaultMaxBody + 1, wantStep: StepSize, wantBytes: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := oversizedBody(tc.size)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := io.WriteString(w, body); err != nil {
					return
				}
			}))
			defer server.Close()

			c, _, _ := clientFor(t, server, map[string][]string{"example.com": {"93.184.216.34"}})
			got := c.Fetch(context.Background(), "example.com")
			if got.FailedAt != tc.wantStep {
				t.Fatalf("FailedAt = %v (%q), want %v", got.FailedAt, got.Detail, tc.wantStep)
			}
			if got.Bytes != tc.wantBytes {
				t.Fatalf("Bytes = %d, want %d", got.Bytes, tc.wantBytes)
			}
			if !reflect.DeepEqual(got.Identity, Identity{}) {
				t.Fatalf("Identity = %+v, want empty", got.Identity)
			}
			if tc.wantStep == StepSize {
				wantDetail := fmt.Sprintf("the document is larger than %d bytes", defaultMaxBody)
				if got.Detail != wantDetail {
					t.Fatalf("Detail = %q, want %q", got.Detail, wantDetail)
				}
				if strings.Contains(got.Detail, sizeSentinel) {
					t.Fatalf("Detail = %q, want no served body bytes", got.Detail)
				}
			}
		})
	}
}

func TestOversizedStreamReturnsAtCap(t *testing.T) {
	chunk := oversizedBody(1024)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			if _, err := io.WriteString(w, chunk); err != nil {
				return
			}
			flusher.Flush()
		}
	}))
	defer server.Close()

	c, _, _ := clientFor(t, server, map[string][]string{"example.com": {"93.184.216.34"}})
	start := time.Now()
	got := c.Fetch(context.Background(), "example.com")
	elapsed := time.Since(start)
	wantDetail := fmt.Sprintf("the document is larger than %d bytes", defaultMaxBody)
	t.Logf("oversized stream returned %s in %v with Bytes=%d and Identity=%+v", got.FailedAt, elapsed, got.Bytes, got.Identity)

	if got.FailedAt != StepSize {
		t.Fatalf("FailedAt = %v (%q), want size; elapsed = %v", got.FailedAt, got.Detail, elapsed)
	}
	if got.Bytes != 0 {
		t.Fatalf("Bytes = %d, want zero", got.Bytes)
	}
	if !reflect.DeepEqual(got.Identity, Identity{}) {
		t.Fatalf("Identity = %+v, want empty", got.Identity)
	}
	if got.Detail != wantDetail {
		t.Fatalf("Detail = %q, want %q", got.Detail, wantDetail)
	}
	if strings.Contains(got.Detail, sizeSentinel) {
		t.Fatalf("Detail = %q, want no served body bytes", got.Detail)
	}
	if elapsed > time.Second {
		t.Fatalf("elapsed = %v, want a prompt return at the cap", elapsed)
	}
}
