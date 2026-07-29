package controller

import (
	"testing"
)

func TestComputeErrorBudgetRemaining(t *testing.T) {
	tests := []struct {
		name         string
		availability float64
		sloTarget    float64
		want         float64
	}{
		{"perfect availability, full budget", 100.0, 99.9, 100.0},
		{"exactly at target, budget exhausted", 99.9, 99.9, 0.0},
		{"half budget used", 99.95, 99.9, 50.0},
		{"budget breached, negative", 99.0, 99.9, -900.0},
		{"100 percent slo target avoids divide by zero", 100.0, 100.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeErrorBudgetRemaining(tt.availability, tt.sloTarget)
			if diff := got - tt.want; diff > 0.001 || diff < -0.001 {
				t.Errorf("computeErrorBudgetRemaining(%v, %v) = %v, want %v",
					tt.availability, tt.sloTarget, got, tt.want)
			}
		})
	}
}
