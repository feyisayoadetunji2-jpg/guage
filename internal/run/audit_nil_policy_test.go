package run

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/posture-sec/guage/internal/model"
	"github.com/posture-sec/guage/internal/rules"
	"github.com/posture-sec/guage/internal/sep1"
	"github.com/posture-sec/guage/internal/source"
)

func TestAuditNilPolicyStopsBeforeAnyFactory(t *testing.T) {
	t.Parallel()
	_, err := Audit(context.Background(), nil, AuditOptions{
		ToolVersion: "test",
		FailOn:      model.SevHigh,
		NewSource: func(endpoint string) (source.Source, error) {
			t.Fatal("NewSource should not be called")
			return nil, nil
		},
		NewIdentity: func() (sep1.Fetcher, error) {
			t.Fatal("NewIdentity should not be called")
			return nil, nil
		},
		Evaluate: func(c rules.Context) []model.Finding {
			t.Fatal("Evaluate should not be called")
			return nil
		},
		Now: func() time.Time {
			t.Fatal("Now should not be called")
			return time.Now()
		},
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err = %v, want ErrUsage", err)
	}
}
