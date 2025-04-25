CREATE TABLE clusters (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    cluster_id INTEGER REFERENCES clusters(id),
    name TEXT NOT NULL,
    identifier TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL
);

CREATE TABLE metrics (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nods(id),
    timestamp TIMESTAMPTZ NOT NULL,
    core_count INTEGER NOT NULL,
) PARTITION BY RANGE (timestamp);

CREATE TABLE daily_summary (
    node_id INTEGER REFERENCES nodes(id),
    date DATE NOT NULL,
    core_count INTEGER NOT NULL,
    total_hours INTEGER NOT NULL,
    PRIMARY KEY (node_id, date, core_count)
);
