package counter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/swfrench/nginx-log-consumer/exporter/counter"

	"google.golang.org/api/monitoring/v3"
)

func TestResetTime(t *testing.T) {
	tMin := time.Now()
	c := counter.NewStatusCounter("foo", &monitoring.MonitoredResource{}, &monitoring.Service{})
	tMax := time.Now()

	if tReset := c.ResetTime(); tReset.Before(tMin) || tReset.After(tMax) {
		t.Errorf("Expected counter ResetTime to be between [%v, %v]; got %v", tMin, tMax, tReset)
	}
}

func TestCreateMetric(t *testing.T) {
	c := counter.NewStatusCounter("foo", &monitoring.MonitoredResource{}, &monitoring.Service{})

	var projectSpec string
	var descriptor *monitoring.MetricDescriptor

	c.CreateMetricCallback = func(p string, d *monitoring.MetricDescriptor) error {
		projectSpec = p
		descriptor = d
		return nil
	}

	if err := c.Create(); err != nil {
		t.Errorf("Create failed with %v", err)
	}

	if want, got := "projects/foo", projectSpec; got != want {
		t.Errorf("Expected CreateMetricCallback to be called with %s, got %s", want, got)
	}

	// Verify a handful of key descriptor fields:

	if want, got := counter.StatusCountMetric, descriptor.Type; got != want {
		t.Errorf("Expected descriptor passed to CreateMetricCallback for metric %s, got %s", want, got)
	}

	if want, got := "CUMULATIVE", descriptor.MetricKind; got != want {
		t.Errorf("Expected descriptor passed to CreateMetricCallback for metric kind %s, got %s", want, got)
	}

	if want, got := "INT64", descriptor.ValueType; got != want {
		t.Errorf("Expected descriptor passed to CreateMetricCallback for metric type %s, got %s", want, got)
	}

	// Now check that error propagation works as intended:

	c.CreateMetricCallback = func(_ string, _ *monitoring.MetricDescriptor) error {
		return fmt.Errorf("This is an error.")
	}

	if err := c.Create(); err == nil {
		t.Errorf("Create should have failed, but did not.")
	}
}

func TestIncrementMetric(t *testing.T) {
	c := counter.NewStatusCounter("foo", &monitoring.MonitoredResource{}, &monitoring.Service{})

	var projectSpec string
	var timeseries *monitoring.CreateTimeSeriesRequest

	// Check basic counter increments:

	callCount := 0

	c.CreateTimeSeriesCallback = func(p string, ts *monitoring.CreateTimeSeriesRequest) error {
		callCount++
		projectSpec = p
		timeseries = ts
		return nil
	}

	newCounts := map[string]int64{
		"200": 1,
		"503": 2,
	}

	tEndMin := time.Now()
	if err := c.Increment(newCounts); err != nil {
		t.Errorf("Increment(%v) failed with: %v", newCounts, err)
	}
	tEndMax := time.Now()

	if want, got := 1, callCount; want != got {
		t.Errorf("Expected CreateTimeSeriesCallback to be called %d times, got %d", want, got)
	}

	if want, got := "projects/foo", projectSpec; got != want {
		t.Errorf("Expected CreateTimeSeriesCallback to be called with %s, got %s", want, got)
	}

	countsSeen := make(map[string]int64)
	for _, ts := range timeseries.TimeSeries {
		if want, got := ts.Metric.Type, counter.StatusCountMetric; got != want {
			t.Errorf("Expected CreateTimeSeriesCallback called with timeseries data for metric %s, got %s", want, got)
		}

		response_code, ok := ts.Metric.Labels["response_code"]
		if !ok {
			t.Errorf("Expected CreateTimeSeriesCallback called with timeseries data labeled with response_code status: %v", ts.Metric.Labels)
		}

		for _, p := range ts.Points {
			if _, exists := countsSeen[response_code]; exists {
				t.Errorf("CreateTimeSeriesCallback called with duplicate points for response code %s", response_code)
			}

			countsSeen[response_code] = *p.Value.Int64Value
			if want, got := c.ResetTime().UTC().Format(time.RFC3339Nano), p.Interval.StartTime; got != want {
				t.Errorf("Expected CreateTimeSeriesCallback called with points covering interval starting at %v, got %v", want, got)
			}

			if minWant, maxWant, got := tEndMin.UTC().Format(time.RFC3339Nano), tEndMax.UTC().Format(time.RFC3339Nano), p.Interval.EndTime; got < minWant || got > maxWant {
				t.Errorf("Expected CreateTimeSeriesCallback called with points covering interval ending between [%v, %v], got %v", minWant, maxWant, got)
			}
		}
	}

	for code, count := range newCounts {
		seen, ok := countsSeen[code]
		if !ok {
			t.Errorf("Expected CreateTimeSeriesCallback called with increment for response code %s, but it is missing", code)
		} else if want, got := count, seen; want != got {
			t.Errorf("Expected CreateTimeSeriesCallback called with count of %d for repsonse code %s, got %d", want, code, got)
		}
	}

	// Call again and make sure accumulation is working:

	newCounts = map[string]int64{
		"200": 2,
		"503": 3,
	}

	callCount = 0

	if err := c.Increment(newCounts); err != nil {
		t.Errorf("Increment(%v) failed with: %v", newCounts, err)
	}

	if want, got := 1, callCount; want != got {
		t.Errorf("Expected CreateTimeSeriesCallback to be called %d times, got %d", want, got)
	}

	for _, ts := range timeseries.TimeSeries {
		response_code, ok := ts.Metric.Labels["response_code"]
		if !ok {
			t.Errorf("Expected CreateTimeSeriesCallback called with timeseries data labeled with response_code status: %v", ts.Metric.Labels)
		}

		for _, p := range ts.Points {
			countsSeen[response_code] = *p.Value.Int64Value
		}
	}

	for code, wantCount := range map[string]int64{
		"200": 3,
		"503": 5,
	} {
		if want, got := wantCount, countsSeen[code]; got != want {
			t.Errorf("Expected CreateTimeSeriesCallback called with accumulated count of %d for repsonse code %s, got %d", want, code, got)
		}
	}

	// Ensure that calling with no delta results in no timeseries writes:

	callCount = 0
	c.CreateTimeSeriesCallback = func(_ string, _ *monitoring.CreateTimeSeriesRequest) error {
		callCount++
		return nil
	}

	if err := c.Increment(map[string]int64{}); err != nil {
		t.Errorf("Increment({}) failed with: %v", err)
	}

	if want, got := 0, callCount; want != got {
		t.Errorf("Expected CreateTimeSeriesCallback to be called %d times, got %d", want, got)
	}

	// Now check that error propagation works as intended:

	c.CreateTimeSeriesCallback = func(_ string, _ *monitoring.CreateTimeSeriesRequest) error {
		return fmt.Errorf("This is an error.")
	}

	if err := c.Increment(map[string]int64{
		"200": 1,
		"500": 2,
	}); err == nil {
		t.Errorf("Increment() should have failed, but did not.")
	}
}
