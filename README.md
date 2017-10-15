# Export metrics from nginx access logs to Stackdriver

A small command-line utility for exporting metrics inferred from nginx access
logs to custom Stackdriver metrics. Currently only supports HTTP response
status code counts, but should be straightforward to extend.

## Requirements

    go get -u cloud.google.com/go/monitoring/apiv3

This should also pull in other deps, like the instance metadata service.

## TODO

* Better unit tests for CloudMonitoringExporter.
