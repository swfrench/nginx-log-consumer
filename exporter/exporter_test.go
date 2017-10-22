package exporter_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/swfrench/nginx-log-consumer/exporter"

	"google.golang.org/api/monitoring/v3"
)

type MockCounter struct {
	resetTime      time.Time
	createCount    int64
	incrementCount int64
	resetTimeCount int64
	counts         map[string]int64
	err            error
}

func (c *MockCounter) Create() error {
	c.createCount += 1
	return c.err
}

func (c *MockCounter) Increment(counts map[string]int64) error {
	c.incrementCount += 1
	c.counts = counts
	return c.err
}

func (c *MockCounter) ResetTime() time.Time {
	c.resetTimeCount += 1
	return c.resetTime
}

func TestBasic(t *testing.T) {
	resource := map[string]string{
		"instance_id": "foo",
		"zone":        "us-central1-a",
	}
	e := exporter.NewCloudMonitoringExporter("foo", resource, &monitoring.Service{})

	c := &MockCounter{
		resetTime: time.Now(),
	}
	e.ReplaceStatusCounter(c)

	if err := e.CreateMetrics(); err != nil {
		t.Fatalf("CreateMetrics failed with %v", err)
	}

	if got, want := c.createCount, int64(1); got != want {
		t.Fatalf("Expected Create to be called %v time(s), got %v", want, got)
	}

	counts := map[string]int64{
		"200": 1,
		"503": 2,
	}

	if err := e.IncrementStatusCounter(counts); err != nil {
		t.Fatalf("IncrementStatusCounter failed with %v", err)
	}

	if got, want := c.counts, counts; !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected equality between status counts passed to IncrementStatusCounter and Increment: got %v vs. %v", got, want)
	}

	if got, want := c.incrementCount, int64(1); got != want {
		t.Fatalf("Expected Increment to be called %v time(s), got %v", want, got)
	}

	if got, want := e.StatusCounterResetTime(), c.resetTime; got != want {
		t.Fatalf("Expected StatusCounterResetTime to return %v, got %v", want, got)
	}

	if got, want := c.resetTimeCount, int64(1); got != want {
		t.Fatalf("Expected ResetTime to be called %v time(s), got %v", want, got)
	}
}

func TestErrorPropagation(t *testing.T) {
	resource := map[string]string{
		"instance_id": "foo",
		"zone":        "us-central1-a",
	}
	e := exporter.NewCloudMonitoringExporter("foo", resource, &monitoring.Service{})

	c := &MockCounter{
		resetTime: time.Now(),
		err:       fmt.Errorf("Test error"),
	}
	e.ReplaceStatusCounter(c)

	if err := e.CreateMetrics(); err == nil {
		t.Fatalf("CreateMetrics should have failed with %v, but it did not", c.err)
	}

	counts := map[string]int64{
		"200": 1,
		"503": 2,
	}

	if err := e.IncrementStatusCounter(counts); err == nil {
		t.Fatalf("IncrementStatusCounter should have failed with %v, but it did not", c.err)
	}
}
