FROM grafana/grafana:latest

COPY provisioning /etc/grafana/provisioning
COPY config.ini /etc/grafana/config.ini
COPY dashboards /var/lib/grafana/dashboards

EXPOSE 3000
