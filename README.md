# cost-metrics-aggregator
This is a deployable component built on golang that can receive payloads from many cost management metrics operators and aggregate the totals. The focus is on summarizing node vCPU utilization for subscription purposes.

## Prerequisites
- Go 1.21+
- PostgreSQL 15+
- OpenShift4.x (for deployment)
- Podman

## Setup
1. Clone the repository:
```bash
git clone https://github.com/chambridge/cost-metrics-aggregator.git
cd cost-metrics-aggregator
```

2. Run database migration:
```bash
psql -d metrics -f internal/db/migrations/0001_init.sql
psql -d metrics -f internal/db/migrations/0002_create_partitions.sql
```

3. Build and run:
```bash
go buld -o server ./cmd/server
./server
```

## Deployment
1. Build the container image:
```bash
podman build -t quay.io/chambridge/cost-metrics-aggregator:latest .
podman push quay.io/chambridge/cost-metrics-aggregator:latest
```

2. Deploy to OpenShift:
```bash
oc apply -f deploy/
```

## Endpoints
- POST /api/ingres/v1/upload: Uplaod a tar.gz file containing manifest.json and node.csv
- GET /api/metrics/v1/ndoes: Query node core count with optional filters (start_date, end_date, cluster_id, cluster_name, node_type).

