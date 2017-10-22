package exporter

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

// StatusCounter implements CounterMetricT for HTTP respone status code counts.
type StatusCounter struct {
	projectSpec string
	resource    *monitoring.MonitoredResource
	service     *monitoring.Service
	counts      map[string]int64
	resetTime   time.Time
}

// NewStatusCounter creats a StatusCounter associated with the provided project
// and MonitoredResource, which will write timeseries values via the provided
// service.
func NewStatusCounter(project string, resource *monitoring.MonitoredResource, service *monitoring.Service) *StatusCounter {
	return &StatusCounter{
		projectSpec: projectResourceSpec(project),
		resource:    resource,
		service:     service,
		counts:      make(map[string]int64),
		resetTime:   time.Now(),
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

	if _, err := c.service.Projects.MetricDescriptors.Create(c.projectSpec, desc).Do(); err != nil {
		return err
	}

	return nil
}

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
			Points:   []*monitoring.Point{
				p,
			},
		}

		timeSeries = append(timeSeries, ts)
	}
	r := &monitoring.CreateTimeSeriesRequest{
		TimeSeries: timeSeries,
	}

	if _, err := c.service.Projects.TimeSeries.Create(c.projectSpec, r).Do(); err != nil {
		return err
	}

	return nil
}

// Increment will accumulate status code count deltas from the supplied map and
// write a new timeseries point.
func (c *StatusCounter) Increment(counts map[string]int64) error {
	for status := range counts {
		if curr, ok := c.counts[status]; ok {
			c.counts[status] = counts[status] + curr
		} else {
			c.counts[status] = counts[status]
		}
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
