# Systemd setup

1. Create an unprivileged user `nginx_log_consumer` with `/usr//sbin/nologin`
   as a shell and read permissions for nginx access logs.
2. Copy the binary to `/usr/sbin/nginx-log-consumer`.
3. Copy `nginx_log_consumer.service` to
   `/etc/systemd/system/nginx_log_consumer.service` and
   `nginx_log_consumer.config` to `/etc/default/nginx_log_consumer`.
4. Run `sudo systemctl enable nginx_log_consumer.service` and `sudo systemctl
   start nginx_log_consumer`.

You may want to examine the unit file and consider adjusting values to your use
case.
