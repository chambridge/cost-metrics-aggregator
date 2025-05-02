# cost-metrics-aggregator
The Cost Metrics Aggregator is a Go-based application for collecting and aggregating cost-related metrics from clusters and nodes, stored in a PostgreSQL database. The focus is on summarizing node vCPU utilization for subscription purposes. It supports partitioned tables for efficient time-series data management and is deployed on OpenShift with automated multi-platform image builds via Quay.io.

## Features
- Collects metrics (e.g., core count) from nodes in clusters.
- Stores data in PostgreSQL with UUID-based identifiers and range-partitioned tables.
- Manages database partitions with automated creation and deletion via CronJobs.
- Deploys on OpenShift with a dedicated PostgreSQL instance and secrets.
- Builds multi-platform container images (`linux/amd64`, `linux/arm64`) using Quay.io.

## Prerequisites
- OpenShift cluster (v4.x) with admin access.
- Quay.io account with permissions to push to `quay.io/chambridge/cost-metrics-aggregator`.
- GitHub repository (`chambridge/cost-metrics-aggregator`) with push access.
- `kubectl` and `podman` installed locally for testing.
- A storage class (e.g., `standard`) available in OpenShift for PostgreSQL persistence.

## Repository Structure
```
.
├── Containerfile
├── go.mod                     # Go module dependencies
├── internal/db/migrations/    # SQL migrations (e.g., 0001_init.sql)
├── scripts/                   # Go scripts for partition management
│   ├── create_partitions.go
│   └── drop_partitions.go
└── deploy/                    # OpenShift manifests
    ├── namespace.yaml
    ├── cost-metrics-db-secret.yaml
    ├── postgres-deployment.yaml
    ├── postgres-service.yaml
    ├── deployment.yaml
    ├── cronjob-create-partitions.yaml
    └── cronjob-drop-partitions.yaml
```

## Database Schema
The database schema (`internal/db/migrations/0001_init.sql`) defines:
- `clusters`: Stores cluster metadata with UUID `id`.
- `nodes`: Stores node metadata with UUID `id`, referencing `clusters`.
- `metrics`: Stores time-series metrics with UUID `id`, partitioned by `timestamp`.
- `daily_summary`: Aggregates daily metrics by `node_id`, `date`, and `core_count`.

The `id` columns use UUIDs (via `gen_random_uuid()`) for unique identification. The `metrics` table is partitioned monthly by `timestamp`.

## Setup
### 1. Clone the Repository
```bash
git clone https://github.com/chambridge/cost-metrics-aggregator.git
cd cost-metrics-aggregator
```

### 2. Build image
```bash
podman build -t quay.io/chambridge/cost-metrics-aggregator:latest .
podman push quay.io/chambridge/cost-metrics-aggregator:latest
```


### 3. Deploy on OpenShift
1. Create the `cost-metrics` namespace:
   ```bash
   kubectl apply -f deploy/namespace.yml
   ```
2. Update contents of the deploy/cost-metrics-db-secret.yml

- Add a base64 encoded password
- Base64 encoded: "postgres://<username>:<password>@postgres:5432/costmetrics"

3. Deploy the PostgreSQL database and secret:
   ```bash
   kubectl apply -f deploy/cost-metrics-db-secret.yml -n cost-metrics
   kubectl apply -f deploy/postgres-deployment.yml -n cost-metrics
   kubectl apply -f deploy/postgres-service.yml -n cost-metrics
   ```

4. Deploy the application:
   ```bash
   kubectl apply -f deploy/deployment.yml -n cost-metrics
   ```

5. Deploy CronJobs for partition management:
   ```bash
   kubectl apply -f deploy/cronjob-create-partitions.yml -n cost-metrics
   kubectl apply -f deploy/cronjob-drop-partitions.yml -n cost-metrics
   ```

### 4. Verify Deployment
1. Check PostgreSQL pod status:
   ```bash
   kubectl get pods -n cost-metrics -l app=postgres
   ```

2. Verify migrations and initial partitions:
   ```bash
   kubectl logs <aggregator-pod-name> -c init-db -n cost-metrics
   ```

3. Connect to the database:
   ```bash
   kubectl exec -it <postgres-pod-name> -n cost-metrics -- psql -U costmetrics -d costmetrics
   \dt+ metrics*  # List partitions
   ```

4. Check CronJob execution:
   ```bash
    kubectl get jobs -n cost-metrics
    kubectl logs <job-pod-name> -n cost-metrics
    ```

## Partition Management
- **Creation**: The `create_partitions.go` script (run by the initContainer and `cronjob-create-partitions`) creates `metrics` table partitions for the next 3 months, named `metrics_YYYY_MM`.
- **Deletion**: The `drop_partitions.go` script (run by `cronjob-drop-partitions`) drops partitions older than 6 months.
- Both CronJobs run monthly on the 1st at midnight (`0 0 1 * *`).

## Endpoints
- POST /api/ingres/v1/upload: Uplaod a tar.gz file containing manifest.json and node.csv
- GET /api/metrics/v1/ndoes: Query node core count with optional filters (start_date, end_date, cluster_id, cluster_name, node_type).

## Troubleshooting
- **Build Failures**: Check Quay.io build logs for missing files or network issues (e.g., `migrate` download).
- **Migration Errors**: Verify `DATABASE_URL` in `cost-metrics-db` secret and PostgreSQL connectivity.
- **CronJob Failures**: Check job logs for script errors or database permissions.

## Contributing
- Submit pull requests to `chambridge/cost-metrics-aggregator`.
- Ensure `internal/db/migrations/` and `scripts/` are updated as needed.
- Test builds locally before pushing to Quay.io.
