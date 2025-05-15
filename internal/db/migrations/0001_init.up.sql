-- Initial schema for cost-metrics-aggregator
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE clusters (
    id UUID PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID REFERENCES clusters(id),
    name TEXT NOT NULL,
    identifier TEXT UNIQUE,
    type TEXT NOT NULL
);

CREATE TABLE node_metrics (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    node_id UUID REFERENCES nodes(id),
    timestamp TIMESTAMPTZ NOT NULL,
    core_count INTEGER NOT NULL,
    cluster_id UUID REFERENCES clusters(id),
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

CREATE TABLE node_daily_summary (
    node_id UUID REFERENCES nodes(id),
    date DATE NOT NULL,
    core_count INTEGER NOT NULL,
    total_hours INTEGER NOT NULL,
    PRIMARY KEY (node_id, date, core_count)
);

CREATE TABLE pods (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id),
    node_id UUID NOT NULL REFERENCES nodes(id),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL,
    component TEXT,
    UNIQUE(name, namespace, cluster_id)
);

CREATE TABLE pod_metrics (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pod_id UUID NOT NULL REFERENCES pods(id),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    pod_usage_cpu_core_seconds DOUBLE PRECISION NOT NULL,
    pod_request_cpu_core_seconds DOUBLE PRECISION NOT NULL,
    node_capacity_cpu_core_seconds DOUBLE PRECISION NOT NULL,
    node_capacity_cpu_cores INTEGER NOT NULL,
    pod_effective_core_seconds DOUBLE PRECISION GENERATED ALWAYS AS (
        GREATEST(pod_usage_cpu_core_seconds, pod_request_cpu_core_seconds)
    ) STORED,
    pod_effective_core_usage DOUBLE PRECISION GENERATED ALWAYS AS (
        CASE
            WHEN node_capacity_cpu_core_seconds > 0
            THEN GREATEST(pod_usage_cpu_core_seconds, pod_request_cpu_core_seconds) / node_capacity_cpu_core_seconds
            ELSE 0
        END
    ) STORED
) PARTITION BY RANGE (timestamp);

CREATE TABLE pod_daily_summary (
    pod_id UUID NOT NULL REFERENCES pods(id),
    date DATE NOT NULL,
    max_cores_used DOUBLE PRECISION NOT NULL,
    total_pod_effective_core_seconds DOUBLE PRECISION NOT NULL,
    total_hours INTEGER NOT NULL,
    PRIMARY KEY (pod_id, date)
);
