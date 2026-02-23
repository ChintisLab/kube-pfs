package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registerOnce sync.Once

	writeLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pfs_write_latency_seconds",
		Help:    "Write latency per component and node",
		Buckets: prometheus.DefBuckets,
	}, []string{"component", "node"})

	readThroughput = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pfs_read_throughput_bytes",
		Help: "Total read throughput in bytes",
	}, []string{"component", "node"})

	iopsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pfs_iops_total",
		Help: "Total IO operations observed",
	}, []string{"component", "node", "op"})

	mdsLockContention = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pfs_mds_lock_contention_seconds",
		Help:    "Observed wait time before MDS lock acquisition",
		Buckets: prometheus.DefBuckets,
	})

	csiOpsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pfs_csi_operations_total",
		Help: "Count of CSI operations by operation and result",
	}, []string{"operation", "result"})

	faultEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pfs_fault_injection_events_total",
		Help: "Fault injection events by action and status",
	}, []string{"action", "status"})

	faultLastEventUnix = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pfs_fault_injection_last_event_unix",
		Help: "Unix timestamp of last fault injection event",
	})
)

func ObserveWriteLatency(component, node string, d time.Duration) {
	writeLatency.WithLabelValues(component, node).Observe(d.Seconds())
}

func AddReadThroughput(component, node string, bytes int) {
	if bytes <= 0 {
		return
	}
	readThroughput.WithLabelValues(component, node).Add(float64(bytes))
}

func IncIOPS(component, node, op string) {
	iopsTotal.WithLabelValues(component, node, op).Inc()
}

func ObserveMDSLockContention(d time.Duration) {
	mdsLockContention.Observe(d.Seconds())
}

func IncCSIOp(operation, result string) {
	csiOpsTotal.WithLabelValues(operation, result).Inc()
}

func RecordFaultEvent(action, status string, when time.Time) {
	faultEventsTotal.WithLabelValues(action, status).Inc()
	faultLastEventUnix.Set(float64(when.Unix()))
}

func StartServer(listenAddr string) *http.Server {
	registerOnce.Do(func() {})
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{Addr: listenAddr, Handler: mux}
	go func() {
		_ = srv.ListenAndServe()
	}()
	return srv
}
