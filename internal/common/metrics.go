package common

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ServiceMetrics struct {
	EventsProcessed prometheus.Counter
	AlertsRaised    *prometheus.CounterVec
}

func NewServiceMetrics(service string) *ServiceMetrics {
	return &ServiceMetrics{
		EventsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krtms_events_processed_total",
			Help: "Number of events processed by service.",
			ConstLabels: prometheus.Labels{
				"service": service,
			},
		}),
		AlertsRaised: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "krtms_alerts_total",
			Help: "Number of alerts raised grouped by severity.",
			ConstLabels: prometheus.Labels{
				"service": service,
			},
		}, []string{"severity"}),
	}
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
