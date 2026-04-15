package core_test

import (
	"testing"

	"github.com/tinywideclouds/go-power-simulator/internal/simulate/core"
	"github.com/tinywideclouds/go-sim-schema/domain"
)

func TestEvaluateCondition_Numbers(t *testing.T) {
	snapshot := core.StateSnapshot{
		"indoor_temp_c": 16.5,
		"time.hour":     float64(14),
	}

	tests := []struct {
		name      string
		condition domain.EngineCondition
		want      bool
	}{
		{
			name: "Less Than - True",
			condition: domain.EngineCondition{
				ContextKey: "indoor_temp_c",
				Operator:   domain.ConditionOperatorLt,
				Value:      "18.0",
			},
			want: true,
		},
		{
			name: "Greater Than - False",
			condition: domain.EngineCondition{
				ContextKey: "indoor_temp_c",
				Operator:   domain.ConditionOperatorGt,
				Value:      "18.0",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.EvaluateCondition(tt.condition, snapshot)
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_Strings(t *testing.T) {
	snapshot := core.StateSnapshot{
		"season": "winter",
	}

	cond := domain.EngineCondition{
		ContextKey: "season",
		Operator:   domain.ConditionOperatorEq,
		Value:      "winter",
	}

	if !core.EvaluateCondition(cond, snapshot) {
		t.Error("Expected string match to return true")
	}
}
