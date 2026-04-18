package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Contagem de requisições HTTP.",
		},
		[]string{"handler", "code", "method"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler", "code", "method"},
	)
)

func wrapHTTP(handlerLabel string, next http.Handler) http.Handler {
	labels := prometheus.Labels{"handler": handlerLabel}
	return promhttp.InstrumentHandlerDuration(
		httpRequestDuration.MustCurryWith(labels),
		promhttp.InstrumentHandlerCounter(
			httpRequestsTotal.MustCurryWith(labels),
			next,
		),
	)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("GET /", wrapHTTP("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]string{"message": "Hello from Go (net/http)"})
	})))

	mux.Handle("GET /time", wrapHTTP("/time", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"server_time": time.Now().UTC().Format(time.RFC3339Nano)})
	})))

	mux.Handle("GET /metrics", promhttp.Handler())

	log.Println("app-go listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
