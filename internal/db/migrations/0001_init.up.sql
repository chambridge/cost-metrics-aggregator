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

CREATE TABLE metrics (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    node_id UUID REFERENCES nodes(id),
    timestamp TIMESTAMPTZ NOT NULL,
    core_count INTEGER NOT NULL,
    cluster_id UUID REFERENCES clusters(id),
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

CREATE TABLE daily_summary (
    node_id UUID REFERENCES nodes(id),
    date DATE NOT NULL,
    core_count INTEGER NOT NULL,
    total_hours INTEGER NOT NULL,
    PRIMARY KEY (node_id, date, core_count)
);
