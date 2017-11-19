package counter

import (
	"fmt"
	"time"

	"google.golang.org/api/monitoring/v3"
)

const (
	// StatusCountMetric is the name of the custom cumulative metric
	// to which status counts are written.
	StatusCountMetric = "custom.googleapis.com/http_response_count"
)

type CreateMetricCallbackT func(string, *monitoring.MetricDescriptor) error
type CreateTimeSeriesCallbackT func(string, *monitoring.CreateTimeSeriesRequest) error

// StatusCounter implements CounterMetricT for HTTP respone status code counts.
type StatusCounter struct {
	projectSpec string
	resource    *monitoring.MonitoredResource
	counts      map[string]int64
	resetTime   time.Time
	// Public for injection from unit tests:
	CreateMetricCallback     CreateMetricCallbackT
	CreateTimeSeriesCallback CreateTimeSeriesCallbackT
}

// NewStatusCounter creats a StatusCounter associated with the provided project
// and MonitoredResource, which will write timeseries values via the provided
// service.
func NewStatusCounter(project string, resource *monitoring.MonitoredResource, service *monitoring.Service) *StatusCounter {
	return &StatusCounter{
		projectSpec: projectResourceSpec(project),
		resource:    resource,
		counts:      make(map[string]int64),
		resetTime:   time.Now(),
		CreateMetricCallback: func(projectSpec string, desc *monitoring.MetricDescriptor) error {
			_, err := service.Projects.MetricDescriptors.Create(projectSpec, desc).Do()
			return err
		},
		CreateTimeSeriesCallback: func(projectSpec string, req *monitoring.CreateTimeSeriesRequest) error {
			_, err := service.Projects.TimeSeries.Create(projectSpec, req).Do()
			return err
		},
	}
}

// ResetTime returns the reset time of the counter metric (i.e. time since
// which counts have been accumulated).
func (c *StatusCounter) ResetTime() time.Time {
	return c.resetTime
}

// Create will create the custom HTTP response status count metric in
// Stackdriver.
func (c *StatusCounter) Create() error {
	desc := &monitoring.MetricDescriptor{
		Type: StatusCountMetric,
		Labels: []*monitoring.LabelDescriptor{
			&monitoring.LabelDescriptor{
				Key:         "response_code",
				ValueType:   "INT64",
				Description: "HTTP status code",
			},
		},
		MetricKind:  "CUMULATIVE",
		ValueType:   "INT64",
		Description: "Cumulative count of HTTP responses by status code.",
	}

	if err := c.CreateMetricCallback(c.projectSpec, desc); err != nil {
		return err
	}

	return nil
}

// write will build a timeseries based on the current cumulative counter values
// and write the result to stackdriver.
func (c *StatusCounter) write() error {
	var timeSeries []*monitoring.TimeSeries
	for status := range c.counts {
		count := c.counts[status]

		p := &monitoring.Point{
			Interval: &monitoring.TimeInterval{
				StartTime: c.resetTime.UTC().Format(time.RFC3339Nano),
				EndTime:   time.Now().UTC().Format(time.RFC3339Nano),
			},
			Value: &monitoring.TypedValue{
				Int64Value: &count,
			},
		}

		ts := &monitoring.TimeSeries{
			Metric: &monitoring.Metric{
				Type: StatusCountMetric,
				Labels: map[string]string{
					"response_code": status,
				},
			},
			Resource: c.resource,
			Points: []*monitoring.Point{
				p,
			},
		}

		timeSeries = append(timeSeries, ts)
	}
	r := &monitoring.CreateTimeSeriesRequest{
		TimeSeries: timeSeries,
	}

	if err := c.CreateTimeSeriesCallback(c.projectSpec, r); err != nil {
		return err
	}

	return nil
}

// Increment will accumulate status code count deltas from the supplied map and
// write a new timeseries point.
func (c *StatusCounter) Increment(counts map[string]int64) error {
	hasDelta := false
	for status, count := range counts {
		if count > 0 {
			hasDelta = true
		}
		if curr, ok := c.counts[status]; ok {
			c.counts[status] = count + curr
		} else {
			c.counts[status] = count
		}
	}

	if !hasDelta {
		return nil
	}

	if err := c.write(); err != nil {
		return err
	}

	return nil
}

// projectResourceSpec properly formats a project ID for use with the monitoring API.
func projectResourceSpec(projectID string) string {
	return fmt.Sprintf("projects/%s", projectID)
}
