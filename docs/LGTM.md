# LGTM Stack Setup Plan

**Goal:** Full observability on the k3s single-node cluster using Grafana's LGTM stack.

## What is LGTM?

| Letter | Tool | Role | Status |
|--------|------|------|--------|
| **L** | Loki | Log aggregation (collect all pod logs) | [ ] Not started |
| **G** | Grafana | Dashboards and UI (already deployed) | [x] Done — `grafana.admondtamang.com.np` |
| **T** | Tempo | Distributed tracing (track requests across services) | [ ] Not started |
| **M** | Mimir | Long-term metrics storage (Prometheus-compatible) | [x] Done — `mimir.admondtamang.com.np` |
| — | Alloy | Unified agent: ships logs + metrics + traces | [ ] Not started |

## Data Flow

```
k8s pods / apps
      │
      ▼
   Alloy (agent, runs as DaemonSet on each node)
      │
      ├──▶ Loki   (logs)
      ├──▶ Mimir  (metrics)
      └──▶ Tempo  (traces)
                │
                ▼
           Grafana (you look at everything here via datasources)
```

**Key concept:** Alloy is the single collection agent. You configure it once and it fans out to all three backends. No Promtail, no Prometheus scraper, no OpenTelemetry Collector needed separately.

---

## Existing State

- `apps/grafana.yaml` → deployed to `monitoring` namespace
- `apps/prometheus.yaml` → deployed to `default` namespace (will be replaced by Mimir)
- No logs, no traces yet

---

## Steps

### Step 1 — Mimir (metrics backend) [ ] Not started

**What:** Replace Prometheus as the long-term metrics store. Mimir is Prometheus-compatible — Grafana queries it the same way. We run it in single-binary mode (one pod, one PVC).

**Why Mimir over plain Prometheus:**
- Prometheus stores data locally with 15-day default retention
- Mimir stores data on a PVC with configurable retention (we'll set 30 days)
- Same PromQL queries work — Grafana can't tell the difference

**Files to create:** `apps/mimir.yaml`

**What you'll learn:** Helm single-binary mode, object storage alternatives, PromQL, Prometheus remote_write

**Key values changes from defaults (all explained in the file):**
- `kafka.enabled: false` — removes Kafka (saves 1 CPU + 1Gi RAM)
- `rollout_operator.enabled: false` — only useful for multi-zone rolling upgrades
- `ingester.replicas: 1` + `zoneAwareReplication.enabled: false` — default is 3 ingesters
- `store_gateway.replicas: 1` + `zoneAwareReplication.enabled: false` — same
- `querier.replicas: 1`, `query_scheduler.replicas: 1` — reduce from 2
- All caches disabled — Memcached adds 4 pods for homelab-negligible gain
- `mimir.structuredConfig.ingest_storage.enabled: false` — critical: disables Kafka mode in Mimir config
- `mimir.structuredConfig.ingester.push_grpc_method_enabled: true` — re-enables classic write path
- `gateway.ingress` (not top-level `ingress`) — mimir-distributed uses an nginx gateway pod

**Checklist:**
- [x] Create `apps/mimir.yaml`
- [ ] Push to GitHub
- [ ] Verify ArgoCD picks it up (`kubectl get applications -n argocd`)
- [ ] Verify pods running — expect ~11 pods in `monitoring` namespace (`kubectl get pods -n monitoring`)
- [ ] Add Mimir as a Prometheus datasource in Grafana (URL: `http://mimir-distributed-gateway.monitoring.svc/prometheus`)
- [ ] Point existing Prometheus at Mimir via remote_write (or remove Prometheus)

---

### Step 2 — Loki (log aggregation) [ ] Not started

**What:** Aggregate logs from every pod in the cluster. Loki indexes only metadata labels (namespace, pod name, app) — not the full log text. This makes it much cheaper than Elasticsearch.

**Why not just `kubectl logs`:**
- `kubectl logs` only shows one pod at a time, and logs disappear when pods restart
- Loki persists logs to a PVC and lets you query across all pods simultaneously
- You can write LogQL queries like: `{namespace="monitoring"} |= "error"`

**Files to create:** `apps/loki.yaml`

**What you'll learn:** LogQL, label-based indexing vs full-text search, log retention

**Checklist:**
- [ ] Create `apps/loki.yaml`
- [ ] Push to GitHub
- [ ] Verify pod is running (`kubectl get pods -n monitoring`)
- [ ] Add Loki as a datasource in Grafana
- [ ] Confirm logs appear in Grafana Explore with a test query

---

### Step 3 — Tempo (distributed tracing) [ ] Not started

**What:** Track a single request as it flows through multiple services. Each service emits a "span" with timing data; Tempo stitches them into a trace with a tree view.

**When it matters:** When you have multiple apps (e.g., n8n calling an API calling a database), Tempo shows you exactly where latency comes from.

**Files to create:** `apps/tempo.yaml`

**What you'll learn:** Traces vs logs vs metrics, OpenTelemetry, span/trace concepts, TraceQL

**Checklist:**
- [ ] Create `apps/tempo.yaml`
- [ ] Push to GitHub
- [ ] Verify pod is running (`kubectl get pods -n monitoring`)
- [ ] Add Tempo as a datasource in Grafana
- [ ] Send a test trace and view it in Grafana Explore

---

### Step 4 — Alloy (unified agent) [ ] Not started

**What:** Alloy (formerly Grafana Agent) runs as a DaemonSet — one pod per node. It scrapes metrics, tails pod logs, and receives traces, then forwards everything to Mimir/Loki/Tempo.

**Why Alloy instead of separate tools:**
- Without Alloy: you need Promtail (logs) + Prometheus (metrics) + OTel Collector (traces) = 3 agents
- With Alloy: one agent, one config file, one thing to debug

**Files to create:** `apps/alloy.yaml`

**What you'll learn:** DaemonSet vs Deployment, Alloy River config language, log scraping, metric scraping

**Checklist:**
- [ ] Create `apps/alloy.yaml` with Loki + Mimir + Tempo forwarding configured
- [ ] Push to GitHub
- [ ] Verify DaemonSet pod is Running on the node (`kubectl get pods -n monitoring -o wide`)
- [ ] Confirm logs flowing into Loki
- [ ] Confirm metrics flowing into Mimir
- [ ] (Optional) Confirm traces flowing into Tempo

---

### Step 5 — Grafana Dashboards [ ] Not started

Wire everything together in Grafana with pre-built dashboards.

**Checklist:**
- [ ] Import Kubernetes cluster dashboard (ID: 15760)
- [ ] Import Loki logs dashboard (ID: 15141)
- [ ] Import Mimir overview dashboard
- [ ] Set up a Grafana alert (e.g., pod crash loop → notification)
- [ ] Link logs ↔ traces in Grafana (correlate a log line to its trace)

---

## Reference Commands

```bash
# Check all apps ArgoCD knows about
kubectl get applications -n argocd

# Watch pods in monitoring namespace
kubectl get pods -n monitoring -w

# Check a pod's logs
kubectl logs -n monitoring <pod-name> --tail=50

# Check ArgoCD app sync status and errors
kubectl describe application <name> -n argocd

# Force ArgoCD to sync immediately (don't wait 180s)
argocd app sync sourceoftruth

# Check PVCs (persistent disk claims)
kubectl get pvc -n monitoring
```

---

## Concepts Learned So Far

- ArgoCD app-of-apps pattern (sourceoftruth watches apps/)
- GitOps flow: push to GitHub → ArgoCD syncs → cluster state matches
- Helm chart deployment via ArgoCD Application objects
- TLS via cert-manager + Let's Encrypt
- k3s Traefik ingress controller
- PVCs and local-path-provisioner storage

---

_Last updated: Step 1 in progress_
