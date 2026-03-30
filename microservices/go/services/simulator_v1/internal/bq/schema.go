package bq

import (
	"time"
)

// MeterReadingRecord represents a single 15-second tick's power draw.
// This is the massive time-series dataset your TinyML will train its predictions on.
type MeterReadingRecord struct {
	Timestamp      time.Time `bigquery:"timestamp"`
	NodeID         string    `bigquery:"node_id"`
	TotalWatts     float64   `bigquery:"total_watts"`
	IndoorTempC    float64   `bigquery:"indoor_temp_c"`
	ActiveEventIDs []string  `bigquery:"active_event_ids"`
}

// EventLogRecord represents a human or environmental trigger.
// This is the traceability dataset your data scientists will use to debug the model.
type EventLogRecord struct {
	EventID     string    `bigquery:"event_id"`
	NodeID      string    `bigquery:"node_id"`
	ScenarioID  string    `bigquery:"scenario_id"`
	ActorID     string    `bigquery:"actor_id"`
	TriggeredAt time.Time `bigquery:"triggered_at"`

	// We can store a JSON string of the exact parameters used (like water volume or duration)
	// so the ML team knows exactly how hard the dice were rolled.
	ParametersJSON string `bigquery:"parameters_json"`
}
