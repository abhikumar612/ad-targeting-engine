package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "delivery_requests_total",
			Help: "Total delivery requests",
		}, []string{"code"},
	)
	Latency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "delivery_request_duration_seconds",
		Help:    "Request latency seconds",
		Buckets: prometheus.DefBuckets,
	})
	InFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "delivery_in_flight",
		Help: "In-flight HTTP requests",
	})
	RequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "delivery_request_errors_total",
			Help: "Total errors by type",
		}, []string{"type"},
	)
)

func init() {
	prometheus.MustRegister(RequestsTotal, Latency, InFlight, RequestErrors)
}

func MetricsHandler() http.Handler { return promhttp.Handler() }

type rec struct {
	http.ResponseWriter
	code int
}

func (r *rec) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func Measure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		InFlight.Inc()
		defer InFlight.Dec()

		rr := &rec{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rr, r)

		Latency.Observe(time.Since(start).Seconds())
		RequestsTotal.WithLabelValues(strconv.Itoa(rr.code)).Inc()
	})
}