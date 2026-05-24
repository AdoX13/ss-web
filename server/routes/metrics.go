package routes

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
)

// Counters exposed on /metrics (Prometheus text format, no external lib).
var (
	mqttMessagesTotal   atomic.Int64
	ocrRequestsTotal    atomic.Int64
	reviewItemsTotal    atomic.Int64
	httpRequestsTotal   atomic.Int64
)

// IncrMQTTMessages increments the MQTT messages counter. Called by broker.
func IncrMQTTMessages() { mqttMessagesTotal.Add(1) }

// IncrOCRRequests increments the OCR request counter.
func IncrOCRRequests() { ocrRequestsTotal.Add(1) }

// IncrReviewItems increments the review items created counter.
func IncrReviewItems() { reviewItemsTotal.Add(1) }

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	httpRequestsTotal.Add(1)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP go_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
	fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP go_memstats_alloc_bytes Bytes allocated and in use\n")
	fmt.Fprintf(w, "# TYPE go_memstats_alloc_bytes gauge\n")
	fmt.Fprintf(w, "go_memstats_alloc_bytes %d\n", ms.Alloc)

	fmt.Fprintf(w, "# HELP mqtt_messages_total Total MQTT messages processed\n")
	fmt.Fprintf(w, "# TYPE mqtt_messages_total counter\n")
	fmt.Fprintf(w, "mqtt_messages_total %d\n", mqttMessagesTotal.Load())

	fmt.Fprintf(w, "# HELP ocr_requests_total Total OCR requests dispatched\n")
	fmt.Fprintf(w, "# TYPE ocr_requests_total counter\n")
	fmt.Fprintf(w, "ocr_requests_total %d\n", ocrRequestsTotal.Load())

	fmt.Fprintf(w, "# HELP review_items_total Total review items created\n")
	fmt.Fprintf(w, "# TYPE review_items_total counter\n")
	fmt.Fprintf(w, "review_items_total %d\n", reviewItemsTotal.Load())

	fmt.Fprintf(w, "# HELP http_requests_total Total HTTP requests to /metrics\n")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
	fmt.Fprintf(w, "http_requests_total %d\n", httpRequestsTotal.Load())
}
