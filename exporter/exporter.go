package exporter

import (
	"time"

	"google.golang.org/api/monitoring/v3"
)

// CounterMetricT provides an interface implemented by all cumulative counter
// metrics. Can be used, for example, to implement mock counters for tests.
type CounterMetricT interface {
	Create() error
	Increment(map[string]int64) error
	ResetTime() time.Time
}

// ExporterT defines the interface implemented by CloudMonitoringExporter. For
// use in mocks.
type ExporterT interface {
	IncrementStatusCounter(map[string]int64) error
	StatusCounterResetTime() time.Time
}

// CloudMonitoringExporter exports metrics collected from nginx access logs to
// custom Stackdriver metrics. Only HTTP response code counts are currently
// supported.
type CloudMonitoringExporter struct {
	statusCounter CounterMetricT
}

// NewCloudMonitoringExporter creates a new CloudMonitoringExporter configured
// to export metrics for the provided project / resource.
func NewCloudMonitoringExporter(project string, resourceLabels map[string]string, service *monitoring.Service) *CloudMonitoringExporter {
	resource := &monitoring.MonitoredResource{
		Labels: resourceLabels,
		Type:   "gce_instance",
	}
	return &CloudMonitoringExporter{
		statusCounter: NewStatusCounter(project, resource, service),
	}
}

// StatusCounterResetTime returnes the reset time of the response status
// counter metric.
func (e *CloudMonitoringExporter) StatusCounterResetTime() time.Time {
	return e.statusCounter.ResetTime()
}

// ReplaceStatusCounter replaces the existing CounterMetricT for the status
// counter metric with a different one. For use in tests.
func (e *CloudMonitoringExporter) ReplaceStatusCounter(counter CounterMetricT) {
	e.statusCounter = counter
}

// CreateMetrics creates the custom Stackdriver metrics written by
// CloudMonitoringExporter. It is assumed that this will have been called at
// least once before the exporter is actually used (e.g. by calling
// IncrementStatusCounts).
func (e *CloudMonitoringExporter) CreateMetrics() error {
	if err := e.statusCounter.Create(); err != nil {
		return err
	}

	return nil
}

// IncrementStatusCounter increments internal HTTP response status counters by
// the provided map of deltas and writes the updated cumulative values to
// Stackdriver.
func (e *CloudMonitoringExporter) IncrementStatusCounter(counts map[string]int64) error {
	if err := e.statusCounter.Increment(counts); err != nil {
		return err
	}

	return nil
}
