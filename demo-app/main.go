package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
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

	// ---- OpenTelemetry Tracing -> Tempo (OTLP HTTP) ----
	tp, err := initTracerProvider()
	if err != nil {
		log.Fatalf("init tracer provider failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}()

	port := getenv("PORT", "8080")

	mux := http.NewServeMux()

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
		delay := time.Duration(100+rand.Intn(1400)) * time.Millisecond
		time.Sleep(delay)
		log.Printf("INFO route=/slow method=%s delay_ms=%d", r.Method, delay.Milliseconds())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "slow ok (%d ms)\n", delay.Milliseconds())
	}))

	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("demo-app listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func initTracerProvider() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// In docker network, Tempo OTLP HTTP endpoint is tempo:4318
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("tempo:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
   	    resource.WithAttributes(
                semconv.ServiceNameKey.String("demo-app"),
    	    ),
	)

	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func instrument(route string, next http.HandlerFunc) http.HandlerFunc {
	tracer := otel.Tracer("demo-app")

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, status: 200}

		// span name contoh: "GET root"
		ctx, span := tracer.Start(r.Context(), r.Method+" "+route)
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", route),
			attribute.String("http.target", r.URL.Path),
		)

		next(rr, r.WithContext(ctx))

		elapsed := time.Since(start).Seconds()

		// metrics
		reqDur.WithLabelValues(r.Method, route).Observe(elapsed)
		reqTotal.WithLabelValues(r.Method, route, strconv.Itoa(rr.status)).Inc()

		// span status
		span.SetAttributes(attribute.Int("http.status_code", rr.status))
		if rr.status >= 500 {
			span.SetStatus(codes.Error, "server error")
		} else {
			span.SetStatus(codes.Ok, "ok")
		}
		span.End()
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
