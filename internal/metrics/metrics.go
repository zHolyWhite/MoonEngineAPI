package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type Histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
}

func NewHistogram(buckets []float64) *Histogram {
	sorted := make([]float64, len(buckets))
	copy(sorted, buckets)
	sort.Float64s(sorted)
	return &Histogram{
		buckets: sorted,
		counts:  make([]uint64, len(sorted)),
	}
}

func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += value
	h.count++
	for i, bound := range h.buckets {
		if value <= bound {
			h.counts[i]++
		}
	}
}

func (h *Histogram) Write(w http.ResponseWriter, name, help string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s histogram\n", name)

	cumulative := uint64(0)
	for i, bound := range h.buckets {
		cumulative = h.counts[i]
		fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", name, bound, cumulative)
	}
	fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", name, h.count)
	fmt.Fprintf(w, "%s_sum %g\n", name, h.sum)
	fmt.Fprintf(w, "%s_count %d\n", name, h.count)
}

type Metrics struct {
	TranslateRequests atomic.Int64
	TranslateCached   atomic.Int64
	TranslateFailed   atomic.Int64
	FlightShared      atomic.Int64
	CacheEntries      func() int
	TranslateDuration *Histogram
}

func New() *Metrics {
	return &Metrics{
		TranslateDuration: NewHistogram([]float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}),
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		fmt.Fprintln(w, "# HELP moonengine_translate_requests_total Total de peticiones de traducción")
		fmt.Fprintln(w, "# TYPE moonengine_translate_requests_total counter")
		fmt.Fprintf(w, "moonengine_translate_requests_total %d\n", m.TranslateRequests.Load())

		fmt.Fprintln(w, "# HELP moonengine_translate_cached_total Resultados servidos desde caché")
		fmt.Fprintln(w, "# TYPE moonengine_translate_cached_total counter")
		fmt.Fprintf(w, "moonengine_translate_cached_total %d\n", m.TranslateCached.Load())

		fmt.Fprintln(w, "# HELP moonengine_translate_failed_total Peticiones de traducción fallidas")
		fmt.Fprintln(w, "# TYPE moonengine_translate_failed_total counter")
		fmt.Fprintf(w, "moonengine_translate_failed_total %d\n", m.TranslateFailed.Load())

		fmt.Fprintln(w, "# HELP moonengine_translate_flight_shared_total Peticiones deduplicadas en vuelo")
		fmt.Fprintln(w, "# TYPE moonengine_translate_flight_shared_total counter")
		fmt.Fprintf(w, "moonengine_translate_flight_shared_total %d\n", m.FlightShared.Load())

		if m.CacheEntries != nil {
			fmt.Fprintln(w, "# HELP moonengine_cache_entries Entradas actuales en caché")
			fmt.Fprintln(w, "# TYPE moonengine_cache_entries gauge")
			fmt.Fprintf(w, "moonengine_cache_entries %d\n", m.CacheEntries())
		}

		m.TranslateDuration.Write(w, "moonengine_translate_duration_ms", "Duración de traducción en milisegundos")
	})
}

func SanitizeLabel(value string) string {
	return strings.ReplaceAll(value, "\"", "'")
}
