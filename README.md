# 🚀 Observability Stack (Prometheus + Grafana + Loki + Tempo + Alertmanager)

A complete observability lab stack built using Docker Compose.

**Includes:**

* 📊 **Prometheus** (metrics)
* 📈 **Grafana** (dashboards & Explore)
* 📜 **Loki** (logs)
* 🔍 **Tempo** (traces)
* 🚨 **Alertmanager** (alert routing)
* 🖥 **Node Exporter** (host metrics)
* 📦 **cAdvisor** (container metrics)
* 🧪 **Demo Go App** (RED + USE testing)

---

## 🧱 Architecture

```text
Demo App
  ├─ Metrics  → Prometheus (/metrics)
  ├─ Traces   → Tempo (OTLP)
  └─ Logs     → Loki (via promtail/docker logging)

Prometheus
  ├─ Scrape: node-exporter, cadvisor, demo-app
  ├─ Rules: alert rules (*.yml)
  └─ Sends alerts → Alertmanager

Grafana
  ├─ Datasource: Prometheus
  ├─ Datasource: Loki
  └─ Datasource: Tempo
```

---

## 📦 Components

### Prometheus

* Scrapes metrics from exporters and the demo app
* Evaluates alert rules
* Sends alerts to Alertmanager

### Alertmanager

* Receives alerts from Prometheus
* Groups / routes notifications (email / Telegram, etc.)

### Grafana

* Dashboards for infra + app
* Explore for Logs (Loki) and Traces (Tempo)

### Loki

* Centralized logging backend

### Tempo

* Distributed tracing backend (query using TraceQL in Grafana Explore)

### Demo App

* Exposes Prometheus metrics:

  * `http_requests_total{method,route,status}`
  * `http_request_duration_seconds_bucket{method,route}`
* Emits OpenTelemetry traces to Tempo
* Endpoints:

  * `GET /` (200)
  * `GET /slow` (random delay)
  * `GET /error` (500)
  * `GET /metrics`

---

## ✅ Requirements

* Docker
* Docker Compose

---

## 🚀 Quick Start

```bash
git clone <your-repo-url>
cd monitoring-stack

docker compose up -d
```

---

## 🌐 Access URLs

| Service      | URL                                            |
| ------------ | ---------------------------------------------- |
| Grafana      | [http://localhost:3000](http://localhost:3000) |
| Prometheus   | [http://localhost:9090](http://localhost:9090) |
| Alertmanager | [http://localhost:9093](http://localhost:9093) |
| Demo App     | [http://localhost:8081](http://localhost:8081) |

**Grafana default login** (if not changed):

```text
admin / admin
```

---

## 🧪 Testing Alerts

> Ensure Prometheus has loaded rules:

* `http://localhost:9090/rules`
* `http://localhost:9090/alerts`

### 1) InstanceDown

Stop a target (example: node-exporter):

```bash
docker stop node-exporter
```

Wait ~1 minute (`for: 1m`) → alert should fire.

Restore:

```bash
docker start node-exporter
```

---

### 2) High CPU Host (USE)

Generate CPU load (adjust `--cpu` to your core count):

```bash
docker run --rm -it alpine sh -lc 'apk add --no-cache stress-ng && stress-ng --cpu 4 --timeout 5m'
```

Wait ~3 minutes (`for: 3m`) → alert should fire.

---

### 3) High 5xx Rate (RED)

Generate 5xx responses:

```bash
for i in $(seq 1 200); do
  curl -s http://localhost:8081/error >/dev/null &
done
wait
```

Wait ~1 minute (`for: 1m`) → alert should fire.

---

## 📊 Useful PromQL

### RED

**Request rate (req/s):**

```promql
sum(rate(http_requests_total[2m]))
```

**5xx error rate (%):**

```promql
100 * sum(rate(http_requests_total{status=~"5.."}[2m]))
  / sum(rate(http_requests_total[2m]))
```

**Latency p95 (seconds):**

```promql
histogram_quantile(
  0.95,
  sum by (le, route) (rate(http_request_duration_seconds_bucket[5m]))
)
```

### USE

**CPU Utilization (%):**

```promql
100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

**Memory Utilization (%):**

```promql
100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))
```

---

## 🔍 Tempo TraceQL Examples

Grafana → **Explore** → datasource **Tempo** → use **TraceQL**.

**All traces:**

```traceql
{}
```

**Filter by service:**

```traceql
{ resource.service.name = "demo-app" }
```

**Slow traces:**

```traceql
{ duration > 500ms }
```

**Errors:**

```traceql
{ status = error }
```

---

## 📁 Project Structure

```text
monitoring-stack/
├── docker-compose.yml
├── prometheus/
│   ├── prometheus.yml
│   └── rules/
├── alertmanager/
│   └── alertmanager.yml
├── grafana/
│   └── provisioning/
├── loki/
├── tempo/
└── demo-app/
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    └── main.go
```

---

## 🎯 Learning Goals

This lab demonstrates:

* Metrics scraping & dashboards
* RED vs USE monitoring methods
* Alerting with Prometheus + Alertmanager
* Centralized logs with Loki
* Distributed tracing with Tempo + TraceQL

---

## 📌 Next Improvements

* SLO / burn-rate alerting
* Service graph and span metrics (Tempo metrics generator)
* Kubernetes deployment (kube-state-metrics, kubelet scrape)
* CI pipeline (lint, build, scan)
* Persistent storage / backup strategy