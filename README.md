# Export metrics from nginx access logs to Stackdriver

A small command-line utility for exporting metrics inferred from nginx access
logs to custom Stackdriver metrics. Currently only supports HTTP response
status code counts, but should be straightforward to extend.

## Requirements

### Dependencies

    go get -u cloud.google.com/go/monitoring/apiv3

This should also pull in other deps, like the instance metadata service.

### Log format

It is expected that nginx has been configured to write logs as json with ISO
8601 timestamps. For example:

    log_format json_combined escape=json '{ "time": "$time_iso8601", '
        '"remote_addr": "$remote_addr", '
        '"remote_user": "$remote_user", '
        '"request": "$request", '
        '"status": "$status", '
        '"body_bytes_sent": "$body_bytes_sent", '
        '"request_time": "$request_time", '
        '"http_referrer": "$http_referer", '
        '"http_user_agent": "$http_user_agent" }';
    access_log /var/log/nginx/access.log json_combined;

As noted above, only the `time` and `status` fields are examined for now.

## TODO

* Better unit tests for CloudMonitoringExporter.
