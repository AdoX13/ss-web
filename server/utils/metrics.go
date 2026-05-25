package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	OcrJobsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ocr_jobs_total",
		Help: "Total number of OCR jobs processed",
	})

	OcrProcessingMsSum = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ocr_processing_ms_sum",
		Help: "Total time spent processing OCR jobs",
	})
	
	OcrProcessingMsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ocr_processing_ms_count",
		Help: "Total count of OCR jobs for latency calculation",
	})
)
