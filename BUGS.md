# Bug & Bad Practice Tracker

Audit of all files in `apps/`. Fix one at a time.
Status: [ ] open  [x] fixed

---

## CRITICAL — Will Break or Is Broken

### [C1] [ ] vault.yaml — HA mode with 3 replicas on a single-node cluster
**File:** `apps/vault.yaml`
**Problem:** `ha.enabled: true` with `replicas: 3`. A single node cannot run 3 Vault pods.
Vault will never reach healthy state — it needs a quorum of 3 nodes to unseal in raft HA mode.
**Fix:** Disable HA, run Vault in standalone mode.
```yaml
# wrong
ha:
  enabled: true
  replicas: 3

# correct for single node
server:
  dev:
    enabled: false
standalone:
  enabled: true
```

### [C2] [ ] _prometheus.yaml — underscore prefix does NOT disable the file
**File:** `apps/_prometheus.yaml`
**Problem:** The `_` prefix looks like "disabled" but ArgoCD picks up ALL `.yaml` files in `apps/`.
Prometheus is actively deployed and exposed publicly with no authentication.
**Fix:** Rename to `prometheus.yaml` (own it) OR delete the file to actually disable it.

---

## SECURITY — Plaintext Passwords in Git

### [S1] [ ] mysql.yaml — credentials committed to Git
**File:** `apps/mysql.yaml`
**Problem:** `rootPassword: "demo"`, `username: "demo"`, `password: "demo"` — in a public repo.
Anyone with repo access can read database credentials.
**Fix:** Use a Kubernetes Secret and reference it, or use a tool like Vault/Sealed Secrets.
Short-term: change from `"demo"` to something non-trivial (still in Git but less obvious).

### [S2] [ ] rabbitmq.yaml — credentials committed to Git
**File:** `apps/rabbitmq.yaml`
**Problem:** `password: "user"` in plaintext in Git.
**Fix:** Same as S1.

### [S3] [ ] redis.yaml — credentials committed to Git
**File:** `apps/redis.yaml`
**Problem:** `password: Password@123` in plaintext in Git.
**Fix:** Same as S1.

---

## BAD PRACTICE — Non-Deterministic Versions

### [B1] [ ] redis.yaml — floating chart version
**File:** `apps/redis.yaml`
**Problem:** `targetRevision: "17.x.x"` — ArgoCD will pick the latest 17.x.x chart.
A chart update can silently break Redis between two deploys with no change in Git.
**Fix:** Pin to an exact version e.g. `targetRevision: "17.11.3"`.

### [B2] [ ] kuma.yaml — floating chart version
**File:** `apps/kuma.yaml` (uptime-kuma)
**Problem:** `targetRevision: 2.x.x` — same issue as B1.
**Fix:** Pin to exact version.

---

## BAD PRACTICE — Wrong or Confusing Config

### [B3] [ ] rabbitmq.yaml — disabled ingress with full ingress config
**File:** `apps/rabbitmq.yaml`
**Problem:** `ingress.enabled: false` but the full ingress block (hostname, TLS, annotations) is still there.
Dead config that is never applied but looks like it should be. Confusing.
**Fix:** Remove the entire ingress block since it's disabled.

### [B4] [ ] mysql.yaml, rabbitmq.yaml, n8n.yaml — `path:` field on Helm chart sources
**File:** `apps/mysql.yaml` (line: `path: mysql`), `apps/rabbitmq.yaml` (line: `path: rabbitmq`), `apps/n8n.yaml` (line: `path: n8n`)
**Problem:** `path:` is only used for Git repo sources (where you point at a directory).
For Helm chart sources (`chart: mysql`, `repoURL: ...`), `path:` is ignored silently.
**Fix:** Remove the `path:` lines from Helm chart sources.

### [B5] [ ] kustomize.yaml — inconsistent cluster reference
**File:** `apps/kustomize.yaml`
**Problem:** Uses `destination.server: https://kubernetes.default.svc` while every other app uses `destination.name: in-cluster`. Both work but it's inconsistent.
**Fix:** Change to `name: in-cluster` to match the rest.

### [B6] [ ] kustomize.yaml — no syncPolicy (not automated)
**File:** `apps/kustomize.yaml`
**Problem:** No `syncPolicy` block. ArgoCD will NOT auto-sync this app — changes to `kustomize/` in Git won't apply automatically.
**Fix:** Add `syncPolicy.automated` block.

### [B7] [ ] kuma.yaml — missing finalizer
**File:** `apps/kuma.yaml`
**Problem:** No `finalizers` block. If you delete this ArgoCD Application, uptime-kuma pods and services will be left running as orphans.
**Fix:** Add `resources-finalizer.argocd.argoproj.io` finalizer.

### [B8] [ ] kustomize.yaml — missing finalizer
**File:** `apps/kustomize.yaml`
**Problem:** Same as B7.
**Fix:** Add finalizer.

---

## BAD PRACTICE — Namespace Organisation

### [B9] [ ] rabbitmq, redis, n8n, vault, nats, prometheus all deploy to `default` namespace
**Problem:** `default` namespace is a dumping ground. All these unrelated apps share one namespace — no isolation, harder to manage RBAC, harder to delete one app cleanly.
**Fix:** Give each app (or category) its own namespace:
- `database` — mysql, redis, rabbitmq
- `monitoring` — prometheus (uptime-kuma already here)
- `messaging` — nats
- `default` — n8n, vault (or their own)

---

## BAD PRACTICE — Sync Wave Order

### [B10] [ ] mysql, rabbitmq, redis are wave 1 — but they should be wave 0
**Problem:** Databases are at `sync-wave: "1"`. Apps that depend on them (n8n needs mysql/redis, etc.) are at wave 0 (default). So apps try to start BEFORE their databases are ready.
`selfHeal: true` masks this — ArgoCD keeps retrying — but it's noisy and slow.
**Fix:** Databases → wave 0. Apps that need them → wave 1.

---

## Summary

| ID  | File | Severity | Status |
|-----|------|----------|--------|
| C1  | vault.yaml | Critical | [ ] |
| C2  | _prometheus.yaml | Critical | [ ] |
| S1  | mysql.yaml | Security | [ ] |
| S2  | rabbitmq.yaml | Security | [ ] |
| S3  | redis.yaml | Security | [ ] |
| B1  | redis.yaml | Bad practice | [ ] |
| B2  | kuma.yaml | Bad practice | [ ] |
| B3  | rabbitmq.yaml | Bad practice | [ ] |
| B4  | mysql/rabbitmq/n8n.yaml | Bad practice | [ ] |
| B5  | kustomize.yaml | Bad practice | [ ] |
| B6  | kustomize.yaml | Bad practice | [ ] |
| B7  | kuma.yaml | Bad practice | [ ] |
| B8  | kustomize.yaml | Bad practice | [ ] |
| B9  | multiple | Bad practice | [ ] |
| B10 | mysql/rabbitmq/redis.yaml | Bad practice | [ ] |
