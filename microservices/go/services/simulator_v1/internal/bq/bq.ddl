-- 1. The High-Volume Time-Series Table
CREATE TABLE `your-project.synthetic_load.meter_readings` (
    timestamp TIMESTAMP NOT NULL,
    node_id STRING NOT NULL,
    total_watts FLOAT64 NOT NULL,
    indoor_temp_c FLOAT64 NOT NULL,
    active_event_ids ARRAY<STRING> -- The magic traceability link
)
PARTITION BY DATE(timestamp)
CLUSTER BY node_id;

-- 2. The Low-Volume Traceability Table
CREATE TABLE `your-project.synthetic_load.event_log` (
    event_id STRING NOT NULL,
    node_id STRING NOT NULL,
    scenario_id STRING NOT NULL,
    actor_id STRING NOT NULL,
    triggered_at TIMESTAMP NOT NULL,
    parameters_json STRING
)
PARTITION BY DATE(triggered_at)
CLUSTER BY node_id, scenario_id;