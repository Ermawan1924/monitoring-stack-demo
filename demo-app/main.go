package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
)

func main() {
	rand.Seed(time.Now().UnixNano())
	prometheus.MustRegister(reqTotal, reqDur)

	port := getenv("PORT", "8080")

	mux := http.NewServeMux()

	// App endpoints
	mux.HandleFunc("/", instrument("root", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("INFO route=/ method=%s msg=ok", r.Method)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	}))

	mux.HandleFunc("/error", instrument("error", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ERROR route=/error method=%s msg=forced_error", r.Method)
		http.Error(w, "forced error", http.StatusInternalServerError)
	}))

	mux.HandleFunc("/slow", instrument("slow", func(w http.ResponseWriter, r *http.Request) {
		// delay 100ms - 1500ms
		delay := time.Duration(100+rand.Intn(1400)) * time.Millisecond
		time.Sleep(delay)
		log.Printf("INFO route=/slow method=%s delay_ms=%d", r.Method, delay.Milliseconds())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "slow ok (%d ms)\n", delay.Milliseconds())
	}))

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("demo-app listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func instrument(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, status: 200}

		next(rr, r)

		elapsed := time.Since(start).Seconds()
		reqDur.WithLabelValues(r.Method, route).Observe(elapsed)
		reqTotal.WithLabelValues(r.Method, route, strconv.Itoa(rr.status)).Inc()
	}
}

type respRecorder struct {
	http.ResponseWriter
	status int
}

func (r *respRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}