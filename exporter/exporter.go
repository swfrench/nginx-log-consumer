package exporter

import (
	"fmt"
	"time"

	"google.golang.org/api/monitoring/v3"
)

const (
	statusCountMetric = "custom.googleapis.com/http_response_count"
)

type ExporterT interface {
	IncrementStatusCounts(map[string]int64) error
	GetResetTime() time.Time
}

type CloudMonitoringExporter struct {
	monitoringService *monitoring.Service
	projectID         string
	resetTime         time.Time
	statusCounts      map[string]int64
	resourceLabels    map[string]string
}

func NewCloudMonitoringExporter(service *monitoring.Service, projectID string, resourceLabels map[string]string) *CloudMonitoringExporter {
	return &CloudMonitoringExporter{
		monitoringService: service,
		projectID:         projectID,
		resetTime:         time.Now(),
		statusCounts:      make(map[string]int64),
		resourceLabels:    resourceLabels,
	}
}

func (e *CloudMonitoringExporter) GetResetTime() time.Time {
	return e.resetTime
}

func (e *CloudMonitoringExporter) writeStatusCount(status string, count int64) error {
	timeseries := monitoring.TimeSeries{
		Metric: &monitoring.Metric{
			Type: statusCountMetric,
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
