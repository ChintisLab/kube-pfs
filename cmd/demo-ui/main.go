package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed static/*
var staticFS embed.FS

type server struct {
	repoRoot      string
	prometheusURL string
}

type faultEvent struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

type benchSummary struct {
	RunID            string  `json:"runId"`
	Path             string  `json:"path"`
	SeqReadMBps      float64 `json:"seqReadMBps"`
	SeqWriteMBps     float64 `json:"seqWriteMBps"`
	RandReadIOPS     float64 `json:"randReadIops"`
	RandWriteIOPS    float64 `json:"randWriteIops"`
	MetadataInfo     string  `json:"metadataInfo"`
	CollectedAt      string  `json:"collectedAt"`
	HasFioSequential bool    `json:"hasFioSequential"`
	HasFioRandom     bool    `json:"hasFioRandom"`
}

type clusterStatus struct {
	Connected           bool              `json:"connected"`
	GeneratedAt         string            `json:"generatedAt"`
	Namespaces          map[string]int    `json:"namespaces"`
	Phases              map[string]int    `json:"phases"`
	Components          map[string]string `json:"components"`
	Error               string            `json:"error,omitempty"`
	ObservabilityHealth string            `json:"observabilityHealth"`
}

func main() {
	var (
		listenAddr = flag.String("listen", ":8088", "demo ui listen address")
		repoRoot   = flag.String("repo-root", ".", "path to repository root")
		promURL    = flag.String("prom-url", envOrDefault("KUBE_PFS_PROM_URL", "http://127.0.0.1:9090"), "prometheus base url")
	)
	flag.Parse()

	s := &server{repoRoot: *repoRoot, prometheusURL: strings.TrimRight(*promURL, "/")}

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("load embedded static files: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/faults", s.handleFaults)
	mux.HandleFunc("/api/benchmarks/latest", s.handleLatestBenchmark)
	mux.HandleFunc("/api/prometheus", s.handlePromQuery)
	mux.HandleFunc("/api/demo/config", s.handleConfig)

	h := withCORS(loggingMiddleware(mux))
	log.Printf("demo ui listening on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, h); err != nil {
		log.Fatalf("serve demo ui: %v", err)
	}
}

func (s *server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"prometheusURL": s.prometheusURL,
		"repoRoot":      s.repoRoot,
	})
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := clusterStatus{
		Connected:           false,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		Namespaces:          map[string]int{},
		Phases:              map[string]int{},
		Components:          map[string]string{},
		ObservabilityHealth: "unknown",
	}

	out, err := runCommand(ctx, "kubectl", "get", "pods", "-A", "-o", "json")
	if err != nil {
		status.Error = err.Error()
		writeJSON(w, http.StatusOK, status)
		return
	}

	items, err := parsePodItems(out)
	if err != nil {
		status.Error = fmt.Sprintf("parse kubectl output: %v", err)
		writeJSON(w, http.StatusOK, status)
		return
	}

	status.Connected = true
	obsReady := 0
	for _, pod := range items {
		status.Namespaces[pod.Namespace]++
		status.Phases[pod.Phase]++

		name := pod.Name
		switch {
		case strings.HasPrefix(name, "kube-pfs-prometheus"):
			status.Components["prometheus"] = pod.Phase
			if pod.Ready {
				obsReady++
			}
		case strings.HasPrefix(name, "kube-pfs-grafana"):
			status.Components["grafana"] = pod.Phase
			if pod.Ready {
				obsReady++
			}
		}
	}

	switch {
	case obsReady >= 2:
		status.ObservabilityHealth = "healthy"
	case obsReady == 1:
		status.ObservabilityHealth = "degraded"
	default:
		status.ObservabilityHealth = "down"
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *server) handleFaults(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.repoRoot, "artifacts", "faults", "timeline.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{"events": []faultEvent{}})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer f.Close()

	events := make([]faultEvent, 0)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev faultEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	if err := sc.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	const maxEvents = 30
	if len(events) > maxEvents {
		events = events[len(events)-maxEvents:]
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *server) handleLatestBenchmark(w http.ResponseWriter, r *http.Request) {
	benchRoot := filepath.Join(s.repoRoot, "artifacts", "bench")
	dirs, err := os.ReadDir(benchRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{"available": false})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	candidates := make([]string, 0)
	for _, d := range dirs {
		if d.IsDir() {
			candidates = append(candidates, d.Name())
		}
	}
	if len(candidates) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	sort.Strings(candidates)
	latest := candidates[len(candidates)-1]
	runPath := filepath.Join(benchRoot, latest)

	summary, err := summarizeBenchmarkRun(runPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	summary.RunID = latest
	summary.Path = runPath
	summary.CollectedAt = time.Now().UTC().Format(time.RFC3339)

	writeJSON(w, http.StatusOK, map[string]any{"available": true, "summary": summary})
}

func (s *server) handlePromQuery(w http.ResponseWriter, r *http.Request) {
	expr := strings.TrimSpace(r.URL.Query().Get("expr"))
	if expr == "" {
		writeErrMsg(w, http.StatusBadRequest, "query param 'expr' is required")
		return
	}

	u, err := url.Parse(s.prometheusURL + "/api/v1/query")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	q := u.Query()
	q.Set("query", expr)
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func summarizeBenchmarkRun(runPath string) (benchSummary, error) {
	s := benchSummary{}

	seqFile := filepath.Join(runPath, "fio-seq.json")
	randFile := filepath.Join(runPath, "fio-rand.json")
	mdtestFile := filepath.Join(runPath, "mdtest.txt")

	if _, err := os.Stat(seqFile); err == nil {
		s.HasFioSequential = true
		readMBps, writeMBps, _, _, err := parseFioSummary(seqFile)
		if err == nil {
			s.SeqReadMBps = readMBps
			s.SeqWriteMBps = writeMBps
		}
	}
	if _, err := os.Stat(randFile); err == nil {
		s.HasFioRandom = true
		_, _, randRead, randWrite, err := parseFioSummary(randFile)
		if err == nil {
			s.RandReadIOPS = randRead
			s.RandWriteIOPS = randWrite
		}
	}
	if blob, err := os.ReadFile(mdtestFile); err == nil {
		text := strings.TrimSpace(string(blob))
		if text == "" {
			s.MetadataInfo = "mdtest output is empty"
		} else {
			lines := strings.Split(text, "\n")
			s.MetadataInfo = lines[len(lines)-1]
		}
	} else {
		s.MetadataInfo = "mdtest output not found"
	}

	return s, nil
}

func parseFioSummary(path string) (readMBps, writeMBps, readIOPS, writeIOPS float64, err error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	var data struct {
		Jobs []struct {
			Read struct {
				BWBytes float64 `json:"bw_bytes"`
				IOPS    float64 `json:"iops"`
			} `json:"read"`
			Write struct {
				BWBytes float64 `json:"bw_bytes"`
				IOPS    float64 `json:"iops"`
			} `json:"write"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(blob, &data); err != nil {
		return 0, 0, 0, 0, err
	}

	for _, job := range data.Jobs {
		readMBps += job.Read.BWBytes / (1024 * 1024)
		writeMBps += job.Write.BWBytes / (1024 * 1024)
		readIOPS += job.Read.IOPS
		writeIOPS += job.Write.IOPS
	}

	readMBps = round2(readMBps)
	writeMBps = round2(writeMBps)
	readIOPS = round2(readIOPS)
	writeIOPS = round2(writeIOPS)
	return readMBps, writeMBps, readIOPS, writeIOPS, nil
}

type podItem struct {
	Namespace string
	Name      string
	Phase     string
	Ready     bool
}

func parsePodItems(blob []byte) ([]podItem, error) {
	var raw struct {
		Items []struct {
			Metadata struct {
				Namespace string `json:"namespace"`
				Name      string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Phase      string `json:"phase"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(blob, &raw); err != nil {
		return nil, err
	}

	out := make([]podItem, 0, len(raw.Items))
	for _, it := range raw.Items {
		p := podItem{
			Namespace: it.Metadata.Namespace,
			Name:      it.Metadata.Name,
			Phase:     it.Status.Phase,
		}
		for _, c := range it.Status.Conditions {
			if c.Type == "Ready" && strings.EqualFold(c.Status, "True") {
				p.Ready = true
				break
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func runCommand(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%v: %s", err, trimmed)
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeErrMsg(w, status, err.Error())
}

func writeErrMsg(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Truncate(time.Millisecond))
	})
}

func round2(v float64) float64 {
	iv, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return iv
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
