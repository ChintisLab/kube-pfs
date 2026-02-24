const byId = (id) => document.getElementById(id);

const state = {
  config: { prometheusURL: "http://127.0.0.1:9090" },
};

async function fetchJSON(path) {
  const res = await fetch(path);
  if (!res.ok) {
    throw new Error(`${path} -> ${res.status}`);
  }
  return res.json();
}

function renderStatus(data) {
  byId("clusterConnectivity").textContent = data.connected
    ? "Connected to Kubernetes"
    : "Cluster not connected";
  byId("obsHealth").textContent = `Observability: ${data.observabilityHealth}`;

  const list = byId("componentList");
  list.innerHTML = "";
  const components = data.components || {};
  const keys = Object.keys(components);
  if (keys.length === 0) {
    const li = document.createElement("li");
    li.textContent = data.error
      ? `Status error: ${data.error}`
      : "No component status available yet.";
    list.appendChild(li);
    return;
  }

  for (const key of keys) {
    const li = document.createElement("li");
    li.innerHTML = `<span>${key}</span><strong>${components[key]}</strong>`;
    list.appendChild(li);
  }
}

function renderTimeline(data) {
  const list = byId("timelineList");
  const empty = byId("timelineEmpty");
  const events = data.events || [];

  list.innerHTML = "";
  if (!events.length) {
    empty.style.display = "block";
    return;
  }
  empty.style.display = "none";
  for (const ev of [...events].reverse()) {
    const li = document.createElement("li");
    li.innerHTML = `
      <div>
        <div><strong>${ev.action}</strong></div>
        <div class="muted">${ev.timestamp}</div>
      </div>
      <div>
        <span class="timeline-status ${ev.status}">${ev.status}</span>
      </div>
    `;
    li.title = ev.detail || "";
    list.appendChild(li);
  }
}

function renderBenchmark(data) {
  if (!data.available || !data.summary) {
    byId("benchRunId").textContent = "No benchmark run found yet.";
    byId("benchSeqRead").textContent = "--";
    byId("benchSeqWrite").textContent = "--";
    byId("benchRandRead").textContent = "--";
    byId("benchRandWrite").textContent = "--";
    byId("benchMetadata").textContent = "Run make benchmark to generate live results.";
    return;
  }

  const s = data.summary;
  byId("benchRunId").textContent = `Latest run: ${s.runId}`;
  byId("benchSeqRead").textContent = formatNumber(s.seqReadMBps);
  byId("benchSeqWrite").textContent = formatNumber(s.seqWriteMBps);
  byId("benchRandRead").textContent = formatNumber(s.randReadIops);
  byId("benchRandWrite").textContent = formatNumber(s.randWriteIops);
  byId("benchMetadata").textContent = s.metadataInfo || "No metadata benchmark info.";
}

async function queryProm(expr) {
  const enc = encodeURIComponent(expr);
  return fetchJSON(`/api/prometheus?expr=${enc}`);
}

function parsePromValue(payload) {
  try {
    const vec = payload?.data?.result;
    if (!Array.isArray(vec) || vec.length === 0) return null;
    const val = Number(vec[0].value?.[1]);
    return Number.isFinite(val) ? val : null;
  } catch {
    return null;
  }
}

async function renderMetrics() {
  const writeP95Q = "histogram_quantile(0.95, sum(rate(pfs_write_latency_seconds_bucket[5m])) by (le))";
  const readTpQ = "sum(rate(pfs_read_throughput_bytes[5m]))";
  const iopsQ = "sum(rate(pfs_iops_total[5m]))";
  const lockP95Q = "histogram_quantile(0.95, sum(rate(pfs_mds_lock_contention_seconds_bucket[5m])) by (le))";

  try {
    const [writeP95, readTp, iopsResp, lockP95] = await Promise.all([
      queryProm(writeP95Q),
      queryProm(readTpQ),
      queryProm(iopsQ),
      queryProm(lockP95Q),
    ]);

    const writeVal = parsePromValue(writeP95);
    const readVal = parsePromValue(readTp);
    const iopsVal = parsePromValue(iopsResp);
    const lockVal = parsePromValue(lockP95);

    const writeMs = writeVal == null ? 0 : writeVal * 1000;
    const readMBps = readVal == null ? 0 : readVal / (1024 * 1024);
    const iops = iopsVal == null ? 0 : iopsVal;
    const lockMs = lockVal == null ? 0 : lockVal * 1000;

    byId("metricWriteP95").textContent = `${formatNumber(writeMs)} ms`;
    byId("metricReadThroughput").textContent = `${formatNumber(readMBps)} MB/s`;
    byId("metricIops").textContent = `${formatNumber(iops)} ops/s`;
    byId("metricLockP95").textContent = `${formatNumber(lockMs)} ms`;

    const noRecentSamples =
      writeVal == null &&
      readVal == null &&
      iopsVal == null &&
      lockVal == null;

    if (noRecentSamples) {
      byId("metricWriteP95").textContent = "--";
      byId("metricReadThroughput").textContent = "--";
      byId("metricIops").textContent = "--";
      byId("metricLockP95").textContent = "--";
      byId("promHint").textContent =
        `Prometheus is reachable, but there are no recent samples yet. Run "make seed-metrics N=20" and refresh in 10-20 seconds.`;
    } else {
      byId("promHint").textContent = `Prometheus source: ${state.config.prometheusURL}`;
    }
  } catch (err) {
    byId("metricWriteP95").textContent = "--";
    byId("metricReadThroughput").textContent = "--";
    byId("metricIops").textContent = "--";
    byId("metricLockP95").textContent = "--";
    byId("promHint").textContent =
      `Prometheus query failed: ${err.message}. I usually fix this by port-forwarding: kubectl -n kube-pfs-observability port-forward svc/kube-pfs-prometheus 9090:9090`;
  }
}

function formatNumber(v) {
  if (v == null || Number.isNaN(v)) return "--";
  return Number(v).toLocaleString(undefined, { maximumFractionDigits: 2 });
}

async function refreshAll() {
  try {
    const [status, faults, bench] = await Promise.all([
      fetchJSON("/api/status"),
      fetchJSON("/api/faults"),
      fetchJSON("/api/benchmarks/latest"),
    ]);
    renderStatus(status);
    renderTimeline(faults);
    renderBenchmark(bench);
  } catch (err) {
    byId("clusterConnectivity").textContent = `Refresh failed: ${err.message}`;
  }

  await renderMetrics();
}

async function bootstrap() {
  try {
    state.config = await fetchJSON("/api/demo/config");
  } catch {
    state.config = { prometheusURL: "http://127.0.0.1:9090" };
  }

  byId("promLink").href = state.config.prometheusURL || "http://127.0.0.1:9090";
  byId("promLink").textContent = "Open Prometheus (localhost:9090)";
  byId("grafanaLink").href = "http://127.0.0.1:3000";
  byId("grafanaLink").textContent = "Open Grafana (localhost:3000)";

  await refreshAll();
  setInterval(refreshAll, 6000);
}

bootstrap();
