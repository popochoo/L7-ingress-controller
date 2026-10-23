package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type IngressMetrics struct {
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestsDuration *prometheus.HistogramVec
	mirrorRequestsTotal  *prometheus.CounterVec
	droppedRequestsTotal *prometheus.CounterVec
}

var (
	instance *IngressMetrics
)

func InitMetrics() *IngressMetrics {
	instance = &IngressMetrics{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ingress_http_requests_total",
				Help: "Total number of HTTP requests proccessed by Ingress",
			},
			[]string{"path", "method", "status"},
		),
		httpRequestsDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ingress_http_request_duration_seconds",
				Help:    "Latency of HTTP requests in seconds",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
			[]string{"path", "method"},
		),
		mirrorRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ingress_mirrored_requests_total",
				Help: "Total number of successfully mirrored requests",
			},
			[]string{"path"},
		),
		droppedRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ingress_dropped_requests_total",
				Help: "Total number of mirrored requests dropped due to full queue",
			},
			[]string{"path"},
		),
	}

	prometheus.MustRegister(instance.httpRequestsTotal)
	prometheus.MustRegister(instance.httpRequestsDuration)
	prometheus.MustRegister(instance.mirrorRequestsTotal)
	prometheus.MustRegister(instance.droppedRequestsTotal)

	return instance
}

func GetMetrics() *IngressMetrics {
	return instance
}

func (m *IngressMetrics) IncRequests(path, method string, status int) {
	m.httpRequestsTotal.WithLabelValues(path, method, strconv.Itoa(status)).Inc()
}

func (m *IngressMetrics) ObserveDuration(path, method string, start time.Time) {
	m.httpRequestsDuration.WithLabelValues(path, method).Observe(float64(time.Since(start)))
}

func (m *IngressMetrics) IncMirrored(path string) {
	m.mirrorRequestsTotal.WithLabelValues(path).Inc()
}

func (m *IngressMetrics) IncDropped(path string) {
	m.droppedRequestsTotal.WithLabelValues(path).Inc()
}

func NewMetricsHandler() http.Handler {
	return promhttp.Handler()
}
