# Systemd setup

1. Create an unprivileged system user `nginx_log_consumer` with read
   permissions for nginx access logs. For example, on a Debian-based
   distribution: `useradd -r nginx_log_consumer -s /usr/sbin/nologin` and
   `usermod -a -G adm nginx_log_consumer`.
2. Copy the binary to `/usr/sbin/nginx-log-consumer`.
3. Copy `nginx_log_consumer.service` to an appropriate location such that
   systemd can find it (e.g. `/etc/systemd/` or `/lib/systemd/`) and copy
   `nginx_log_consumer.config` to `/etc/default/nginx_log_consumer`.
4. Run `sudo systemctl enable nginx_log_consumer.service` and `sudo systemctl
   start nginx_log_consumer`.

You may want to examine the unit file and consider adjusting values to your use
case.
