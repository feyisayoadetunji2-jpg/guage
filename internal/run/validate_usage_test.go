package run

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/posture-sec/guage/internal/model"
)

func TestInvalidJobReturnsErrUsageAndNoReport(t *testing.T) {
	t.Parallel()
	valid := func() Job {
		return baseJob(&fakeSource{info: model.NetworkInfo{Passphrase: testnet, Ledger: 1}})
	}
	cases := []struct {
		name string
		edit func(*Job)
	}{
		{"no network", func(j *Job) { j.Network = "" }},
		{"no endpoint", func(j *Job) { j.Endpoints = nil }},
		{"the same endpoint twice", func(j *Job) { j.Endpoints = append(j.Endpoints, j.Endpoints[0]) }},
		{"more endpoints than a comparison can print", func(j *Job) {
			for i := 0; i <= maxEndpoints; i++ {
				j.Endpoints = append(j.Endpoints, fmt.Sprintf("https://extra-%d.example", i))
			}
		}},
		{"no accounts", func(j *Job) { j.Accounts = nil }},
		{"negative concurrency", func(j *Job) { j.Concurrency = -1 }},
		{"unknown failure threshold", func(j *Job) { j.FailOn = model.Severity(9) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			job := valid()
			tt.edit(&job)
			report, err := Run(context.Background(), job)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("err = %v, want ErrUsage", err)
			}
			if !reflect.DeepEqual(report, model.Report{}) {
				t.Fatalf("report = %+v, want empty report", report)
			}
		})
	}
}
