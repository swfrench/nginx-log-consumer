package exporter

import (
	"fmt"
	"time"

	"google.golang.org/api/monitoring/v3"
)

const (
	// CustomStatusCountMetric is the name of the custom cumulative metric
	// to which status counts are written.
	CustomStatusCountMetric = "custom.googleapis.com/http_response_count"
)

// ExporterT defines the interface implemented by CloudMonitoringExporter. For
// use in mocks.
type ExporterT interface {
	IncrementStatusCounts(map[string]int64) error
	GetResetTime() time.Time
}

// CloudMonitoringExporter exports various metrics collected from nginx access
// logs to custom Stackdriver metrics. Only HTTP response code counts are
// currently supported.
// Note: CloudMonitoringExporter assumes that the cumulative response status
// count metric already exists. See CustomStatusCountMetric.
type CloudMonitoringExporter struct {
	monitoringService *monitoring.Service
	projectID         string
	resetTime         time.Time
	statusCounts      map[string]int64
	resourceLabels    map[string]string
}

// NewCloudMonitoringExporter creates a new CloudMonitoringExporter configured
// to export metrics for the provided project / resource.
func NewCloudMonitoringExporter(service *monitoring.Service, projectID string, resourceLabels map[string]string) *CloudMonitoringExporter {
	return &CloudMonitoringExporter{
		monitoringService: service,
		projectID:         projectID,
		resetTime:         time.Now(),
		statusCounts:      make(map[string]int64),
		resourceLabels:    resourceLabels,
	}
}

// GetResetTime returns last reset time of cumulative counter metrics.
func (e *CloudMonitoringExporter) GetResetTime() time.Time {
	return e.resetTime
}

func (e *CloudMonitoringExporter) writeStatusCount(status string, count int64) error {
	timeseries := monitoring.TimeSeries{
		Metric: &monitoring.Metric{
			Type: CustomStatusCountMetric,
			Labels: map[string]string{
				"response_code": status,
			},
		},
		Resource: &monitoring.MonitoredResource{
			Labels: e.resourceLabels,
			Type:   "gce_instance",
		},
		Points: []*monitoring.Point{
			{
				Interval: &monitoring.TimeInterval{
					StartTime: e.resetTime.UTC().Format(time.RFC3339Nano),
					EndTime:   time.Now().UTC().Format(time.RFC3339Nano),
				},
				Value: &monitoring.TypedValue{
					Int64Value: &count,
				},
			},
		},
	}

	createTimeseriesRequest := monitoring.CreateTimeSeriesRequest{
		TimeSeries: []*monitoring.TimeSeries{&timeseries},
	}

	_, err := e.monitoringService.Projects.TimeSeries.Create(fmt.Sprintf("projects/%s", e.projectID), &createTimeseriesRequest).Do()
	if err != nil {
		return err
	}

	return nil
}

// IncrementStatusCounts increments internal HTTP response status counters by
// the provided map of deltas and writes the updated cumulative values to
// Stackdriver.
func (e *CloudMonitoringExporter) IncrementStatusCounts(counts map[string]int64) error {
	for status := range counts {
		if curr, ok := e.statusCounts[status]; ok {
			e.statusCounts[status] = counts[status] + curr
		} else {
			e.statusCounts[status] = counts[status]
		}
	}
	for status := range e.statusCounts {
		if err := e.writeStatusCount(status, e.statusCounts[status]); err != nil {
			return err
		}
	}

	return nil
}
