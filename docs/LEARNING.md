# Kubernetes Learning Notes

Live notes from working on this cluster. Concepts explained through real files in this repo.

---

## ArgoCD Application Object

Every file in `apps/` is an ArgoCD `Application` object. It is an *instruction*, not the app itself.

```yaml
apiVersion: argoproj.io/v1alpha1   # ArgoCD's API group
kind: Application                   # ArgoCD's custom object type
metadata:
  name: mysql                       # object's unique name
  namespace: argocd                 # where the instruction lives
spec:
  project: default                  # security boundary (default = no restrictions)
  destination:
    namespace: database             # where the app's resources get deployed
    name: in-cluster                # which cluster (in-cluster = this k3s node)
```

### Two namespaces — different roles

| Field | Namespace | What lives there |
|---|---|---|
| `metadata.namespace` | `argocd` | The Application object (just instructions/YAML) |
| `destination.namespace` | `database`, `monitoring`, etc. | Actual pods, services, PVCs |

---

## Where MySQL Data Actually Lives

The `Application` object in `argocd` namespace stores **zero MySQL data** — it's just config.

Real MySQL resources live in the `destination.namespace` (`database`):

```
argocd namespace
└── Application/mysql        ← instruction only (YAML config)

database namespace
├── Pod/mysql-0              ← running MySQL process
├── Service/mysql            ← network access
├── Secret/mysql             ← passwords
└── PersistentVolumeClaim    ← actual database files (tables, rows, indexes)
        │
        ▼
    /var/lib/rancher/k3s/storage/   ← real files on node disk
```

A **PersistentVolumeClaim (PVC)** reserves disk space on the node. MySQL writes its data there just like a normal MySQL install writes to `/var/lib/mysql`.

Check it: `kubectl get pvc -n database`

> **Important:** If you delete the ArgoCD Application, the PVC can survive — but `prune: true` will delete it too. Handle database deletions carefully.

---

## Kubernetes Object Structure

Every Kubernetes object (Pod, Service, Deployment, Application...) has this same header:

```yaml
apiVersion: <who-defined-this>/<version>
kind: <type-of-object>
metadata:
  name: <unique-name>
  namespace: <which-folder>
spec:
  ...  # the actual config
```

Core k8s objects use `apiVersion: v1`. Custom types added by tools use a domain prefix:
- `argoproj.io/v1alpha1` — ArgoCD's types (Application, AppProject)
- `cert-manager.io/v1` — cert-manager's types (Certificate, ClusterIssuer)

---

## Finalizers

A **finalizer** is a lock on a Kubernetes object that forces cleanup before deletion.

```yaml
finalizers:
  - resources-finalizer.argocd.argoproj.io
```

When you `kubectl delete application mysql`:
1. Kubernetes sets `deletionTimestamp` (marks it for deletion, doesn't delete yet)
2. ArgoCD sees the timestamp → deletes all MySQL resources (pods, services, PVCs)
3. ArgoCD removes the finalizer lock
4. Kubernetes fully deletes the Application object

Without this finalizer, the Application object would vanish but MySQL pods would keep running as orphans.

> Do NOT rename this finalizer — ArgoCD's controller looks for this exact string.

---

## Sync Waves

Controls deployment **order** when multiple apps sync at once.

```yaml
annotations:
  argocd.argoproj.io/sync-wave: "1"
```

ArgoCD deploys wave 0 first, waits until healthy, then wave 1, etc. Default is wave 0.

```
Wave 0  →  deploy, wait for healthy
Wave 1  →  deploy, wait for healthy
Wave 2  →  deploy
```

In this repo: `mysql`, `rabbitmq`, `redis` are wave 1. Everything else is wave 0 (no annotation). Ideally databases should be wave 0 and apps that depend on them wave 1+.

---

## Deleting an App (GitOps Way)

You cannot just delete from the cluster — Git is the source of truth. ArgoCD will recreate it.

Full clean delete flow:
```bash
# 1. Stop ArgoCD from fighting you during deletion
kubectl patch application <name> -n argocd --type=merge \
  -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'

# 2. Delete the ArgoCD Application (finalizer triggers resource cleanup)
kubectl delete application <name> -n argocd

# 3. Remove from Git so ArgoCD doesn't recreate it
git rm apps/<name>.yaml
git commit -m "chore: remove <name>"
git push
```

---

## LoadBalancer + Klipper (k3s)

k3s uses a built-in load balancer called **Klipper**. It binds service ports directly on the node.

**Rule:** ALL ports in a LoadBalancer service must be free on the node. If even one conflicts, the entire service gets `EXTERNAL-IP: <pending>`.

Port 443 is held by Traefik (k3s built-in ingress). Any LoadBalancer service that also requests port 443 will stay pending.

Fix: explicitly override conflicting ports in Helm values. Removing a value from `values.yaml` falls back to the chart default — you must explicitly set a different value.

```yaml
# values.yaml — ArgoCD example
server:
  service:
    servicePortHttp: 9999
    servicePortHttps: 9998   # must explicitly override — default is 443 (conflicts with Traefik)
```

---

## Helm Values Gotcha

Removing a key from `values.yaml` does NOT disable it — it falls back to the chart's built-in default.

```
Your values.yaml          Chart default
──────────────────        ──────────────────────
servicePortHttp: 9999  →  overrides default 80   ✓
servicePortHttps: (removed) → falls back to 443  ← conflict!
servicePortHttps: 9998  →  overrides default 443 ✓
```

Always check chart defaults with: `helm show values <repo>/<chart>`
