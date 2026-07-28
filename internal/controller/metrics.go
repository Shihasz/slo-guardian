package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	labelPolicy    = "policy"
	labelNamespace = "namespace"
)

var (
	availabilityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sloguardian_availability_percent",
			Help: "Current measured availability percentage for an SLOPolicy target",
		},
		[]string{labelPolicy, labelNamespace},
	)

	errorBudgetGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sloguardian_error_budget_remaining_percent",
			Help: "Remaining error budget percentage for an SLOPolicy target",
		},
		[]string{labelPolicy, labelNamespace},
	)

	checksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sloguardian_checks_total",
			Help: "Total number of health checks performed",
		},
		[]string{labelPolicy, labelNamespace, "result"},
	)

	remediationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sloguardian_remediations_total",
			Help: "Total number of remediation actions taken",
		},
		[]string{labelPolicy, labelNamespace, "action"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		availabilityGauge,
		errorBudgetGauge,
		checksTotal,
		remediationsTotal,
	)
}
