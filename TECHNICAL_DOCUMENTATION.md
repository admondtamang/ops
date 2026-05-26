# Kubernetes Learning Guide & Ops Runbook

> Your personal cluster: k3s v1.34 on a single node (`zmond`), running at `https://127.0.0.1:6443`

---

## Table of Contents

1. [What Is Kubernetes and What Is k3s?](#1-what-is-kubernetes-and-what-is-k3s)
2. [Docker vs k3s — How They Coexist](#2-docker-vs-k3s--how-they-coexist)
3. [Your Cluster At a Glance](#3-your-cluster-at-a-glance)
4. [Core Kubernetes Concepts](#4-core-kubernetes-concepts)
5. [Essential kubectl Commands](#5-essential-kubectl-commands)
6. [Your Repository Structure](#6-your-repository-structure)
7. [Your GitOps Architecture (ArgoCD)](#7-your-gitops-architecture-argocd)
8. [Step-by-Step: Bring the Cluster Back to Full State](#8-step-by-step-bring-the-cluster-back-to-full-state)
9. [Day-to-Day Operations](#9-day-to-day-operations)
10. [Troubleshooting Cheatsheet](#10-troubleshooting-cheatsheet)
11. [Glossary](#11-glossary)

---

## 1. What Is Kubernetes and What Is k3s?

**Kubernetes** is a system that runs and manages containers across one or more machines. Instead of running `docker run` manually, you write a YAML file describing what you want (e.g., "I want 3 copies of nginx running"), and Kubernetes figures out how to make that happen and keeps it that way.

**k3s** is a lightweight, single-binary version of Kubernetes made by Rancher. It is 100% real Kubernetes — just stripped of cloud-specific stuff and packaged for easier installation. It is what you have installed.

```
Standard Kubernetes → complex, multi-component install (meant for big cloud clusters)
k3s               → everything in one binary, ideal for homelab / single machine
```

k3s bundles these components automatically:
- **Traefik** — ingress controller (routes external HTTP/HTTPS traffic into the cluster)
- **CoreDNS** — DNS inside the cluster (pods find each other by name)
- **local-path-provisioner** — automatic disk storage for pods
- **metrics-server** — lets you run `kubectl top`

---

## 2. Docker vs k3s — How They Coexist

This is the most important thing to understand to avoid confusion.

### They use DIFFERENT container runtimes

| Tool | Runtime | Images stored in |
|---|---|---|
| Docker | Docker daemon (`dockerd`) | `/var/lib/docker` |
| k3s | containerd | `/var/lib/rancher/k3s` |

**They do NOT share images.** If you `docker build` an image, k3s cannot see it automatically.

### Rules to avoid conflicts

1. **Never run `docker run` for things you want Kubernetes to manage.** Let kubectl/argocd do it.
2. **Use Docker only for building images locally**, then push to a registry (Docker Hub, GHCR).
3. **If you need a locally-built image in k3s** without a registry, import it:
   ```bash
   docker save myimage:latest | k3s ctr images import -
   ```
4. **k3s does NOT use Docker.** Uninstalling Docker will not break your cluster.

### Why your cluster is fine with Docker installed

k3s started its own containerd process separately. Docker runs its own daemon. They do not talk to each other. As long as you are not trying to use a locally-built Docker image directly in k3s, there is no conflict.

---

## 3. Your Cluster At a Glance

```
Cluster: k3s v1.34.4+k3s1
Node:    zmond (single-node, control-plane + worker combined)
Status:  Running (started via systemd, starts on boot)
API:     https://127.0.0.1:6443
Ingress: Traefik on 192.168.1.64 (ports 80 and 443)
Domain:  *.admondtamang.com.np
```

### What is currently running (as of May 2026)

```
NAMESPACE     COMPONENT                    STATUS
kube-system   traefik                      Running  ← handles all HTTP/HTTPS routing
kube-system   coredns                      Running  ← DNS inside the cluster
kube-system   local-path-provisioner       Running  ← automatic storage
kube-system   metrics-server               Running  ← kubectl top
argocd        (nothing — needs deploying)  ← see Section 8
```

### Applications defined but not deployed

These live in `apps/` and will be deployed once ArgoCD is running:

| File | App | URL |
|---|---|---|
| `apps/grafana.yaml` | Grafana (monitoring dashboards) | grafana.admondtamang.com.np |
| `apps/n8n.yaml` | n8n (workflow automation) | n8n.admondtamang.com.np |
| `apps/kuma.yaml` | Uptime Kuma (status page) | — |
| `apps/mysql.yaml` | MySQL database | — |
| `apps/_prometheus.yaml` | Prometheus (metrics) | — |
| `apps/rabbitmq.yaml` | RabbitMQ (message queue) | — |
| `apps/redis.yaml` | Redis (cache) | — |
| `apps/vault.yaml` | HashiCorp Vault (secrets) | — |

---

## 4. Core Kubernetes Concepts

### The Mental Model

Think of Kubernetes as a **desired state machine**. You tell it what you want, and it continuously works to make reality match what you wrote.

```
You write YAML  →  kubectl apply  →  Kubernetes makes it happen  →  Kubernetes keeps it that way
```

### Pod

The smallest deployable unit. A pod wraps one or more containers. Think of it as a single instance of your app.

```yaml
# A simple pod
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
    - name: app
      image: nginx:latest
      ports:
        - containerPort: 80
```

```bash
kubectl run my-app --image=nginx    # create a pod fast (imperative)
kubectl get pods                    # list pods
kubectl describe pod my-app         # detailed info + events
kubectl logs my-app                 # see stdout of the container
kubectl exec -it my-app -- bash     # shell into the pod
kubectl delete pod my-app           # delete it
```

### Deployment

A Deployment manages pods. It ensures N replicas are always running and handles rolling updates.

```bash
kubectl create deployment nginx --image=nginx --replicas=2   # create
kubectl get deployments                                       # list
kubectl scale deployment nginx --replicas=3                  # scale up
kubectl rollout status deployment nginx                       # watch rollout
kubectl rollout undo deployment nginx                         # roll back
```

### Service

Pods get a new IP every time they restart. A Service gives you a stable IP/DNS name to reach pods.

```
Types:
  ClusterIP    → only reachable inside the cluster (default)
  NodePort     → accessible on the node's IP at a port (30000-32767)
  LoadBalancer → gets an external IP (used by Traefik in your setup)
```

```bash
kubectl expose deployment nginx --port=80 --type=ClusterIP   # create service
kubectl get services                                          # list
```

### Namespace

A way to divide the cluster into isolated sections. Your cluster uses:
- `kube-system` — Kubernetes internals (Traefik, CoreDNS, etc.)
- `argocd` — ArgoCD lives here
- `default` — where your apps land if no namespace is specified
- `cert-manager` — the TLS certificate manager

```bash
kubectl get pods -n argocd           # pods in argocd namespace
kubectl get pods --all-namespaces    # pods everywhere (short: -A)
```

### Ingress

Rules that tell Traefik how to route HTTP/HTTPS traffic from the outside world to a Service.

```yaml
# Example: route grafana.admondtamang.com.np → grafana service port 80
kind: Ingress
spec:
  rules:
    - host: grafana.admondtamang.com.np
      http:
        paths:
          - path: /
            backend:
              service:
                name: grafana
                port:
                  number: 80
```

### ConfigMap and Secret

- **ConfigMap** — non-sensitive configuration (env vars, config files)
- **Secret** — sensitive data (passwords, tokens) — stored base64-encoded

```bash
kubectl get configmaps -A
kubectl get secrets -A
```

### Persistent Volume (PV) and Persistent Volume Claim (PVC)

Pods are stateless by default — data is lost when a pod dies. A PVC requests durable storage.

In your cluster, `local-path-provisioner` automatically creates storage on the node's disk when a pod requests it via a PVC.

---

## 5. Essential kubectl Commands

### Cluster overview

```bash
kubectl cluster-info                        # cluster endpoint
kubectl get nodes                           # node status
kubectl top nodes                           # CPU/memory usage
kubectl get all -A                          # everything everywhere
```

### Working with resources

```bash
# Get (list)
kubectl get pods
kubectl get pods -n argocd
kubectl get pods -A                         # all namespaces
kubectl get pods -o wide                    # more columns (node, IP)
kubectl get pods -w                         # watch for changes

# Describe (detailed + events)
kubectl describe pod <name>
kubectl describe node zmond

# Logs
kubectl logs <pod-name>
kubectl logs <pod-name> -f                  # follow (like tail -f)
kubectl logs <pod-name> --previous          # logs from crashed container

# Shell into a pod
kubectl exec -it <pod-name> -- bash
kubectl exec -it <pod-name> -- sh           # if bash not available

# Apply / Delete
kubectl apply -f file.yaml                  # create or update
kubectl delete -f file.yaml                 # delete what's in the file
kubectl apply -k ./kustomize                # apply a Kustomize directory
```

### Helm (used to install ArgoCD)

```bash
helm list -A                                # see all helm releases
helm list -n argocd                         # helm releases in argocd namespace
helm upgrade --install <name> <chart>       # install or upgrade
```

---

## 6. Your Repository Structure

```
ops/
├── argo-config.yaml          ← Root ArgoCD app ("sourceoftruth") + ArgoCD Ingress
├── apps/                     ← ArgoCD Application definitions (one per app)
│   ├── grafana.yaml
│   ├── n8n.yaml
│   ├── kuma.yaml
│   ├── mysql.yaml
│   ├── _prometheus.yaml
│   ├── rabbitmq.yaml
│   ├── redis.yaml
│   └── vault.yaml
├── bootstrap/
│   └── argocd/
│       ├── install.sh        ← Helm script to install ArgoCD
│       └── values.yaml       ← ArgoCD configuration (version, resource limits, RBAC)
├── kustomize/
│   ├── kustomization.yaml    ← Installs cert-manager from upstream + local issuers
│   ├── cert-issuer-staging.yaml
│   └── cert-issuer-production.yaml
└── graveyard/                ← Old configs, kept for reference
```

---

## 7. Your GitOps Architecture (ArgoCD)

### What is GitOps?

GitOps means Git is the **single source of truth**. You never `kubectl apply` things manually in production — instead, you push a YAML change to GitHub, and ArgoCD detects it and applies it to the cluster automatically.

### App of Apps Pattern

```
GitHub (ops.git)
    └── ArgoCD watches this repo
            └── argo-config.yaml defines "sourceoftruth" app
                    └── sourceoftruth watches apps/ directory
                            ├── grafana.yaml    → deploys Grafana
                            ├── n8n.yaml        → deploys n8n
                            └── kuma.yaml       → deploys Kuma
                                    ...
```

**"sourceoftruth"** is the root application. It tells ArgoCD to watch the `apps/` directory of your Git repo. Every YAML file in `apps/` is an ArgoCD Application. When you add or change a file there and push to GitHub, ArgoCD picks it up and deploys it.

### Traefik Ingress Flow

```
Internet → 192.168.1.64 (your router/node IP)
    → Traefik (port 80/443 in kube-system)
        → Reads Ingress rules
            → Routes to Service in cluster
                → Pod runs your app
```

### cert-manager TLS Flow

```
You add annotation:  cert-manager.io/cluster-issuer: letsencrypt-prod
cert-manager sees it → calls Let's Encrypt API → proves domain ownership via HTTP
Let's Encrypt issues certificate → stored as a Secret → Traefik uses it for HTTPS
```

---

## 8. Step-by-Step: Bring the Cluster Back to Full State

Your k3s is running. Your apps are defined. But ArgoCD is not deployed. Here is the order to get everything back.

### Step 1 — Verify the cluster is healthy

```bash
kubectl get nodes
# Expected: zmond   Ready   control-plane,master

kubectl get pods -n kube-system
# Expected: traefik, coredns, metrics-server, local-path-provisioner all Running
```

### Step 2 — Deploy ArgoCD

```bash
cd /home/zmond/Work/ops/bootstrap/argocd
./install.sh
```

This runs Helm to install ArgoCD in the `argocd` namespace. It will take 1-2 minutes. Watch it:

```bash
kubectl get pods -n argocd -w
# Wait until all pods show Running or Completed
```

### Step 3 — Apply cert-manager (TLS certificates)

```bash
cd /home/zmond/Work/ops
kubectl apply -k ./kustomize
```

Wait for cert-manager to be ready:

```bash
kubectl get pods -n cert-manager -w
```

### Step 4 — Apply the root ArgoCD config

```bash
kubectl apply -f argo-config.yaml
```

This creates the `sourceoftruth` application and the ArgoCD Ingress. ArgoCD will then auto-sync and deploy everything in `apps/`.

### Step 5 — Watch apps deploy

```bash
kubectl get pods -A -w
```

### Step 6 — Access ArgoCD UI

Open: `https://argo.admondtamang.com.np`

Get the initial admin password:

```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo
```

Login with `admin` and that password. Change it immediately after.

---

## 9. Day-to-Day Operations

### Deploy a new app

1. Create `apps/myapp.yaml` with an ArgoCD Application definition (copy `apps/grafana.yaml` as a template)
2. `git add apps/myapp.yaml && git commit -m "feat: add myapp" && git push`
3. ArgoCD detects the change and deploys it. Done.

### Update an app (e.g., change chart version)

1. Edit the file in `apps/`
2. `git commit && git push`
3. ArgoCD auto-syncs (up to 180 seconds, per your `timeout.reconciliation` setting)

### Force a sync immediately

```bash
# via ArgoCD CLI (if installed)
argocd app sync sourceoftruth

# or via kubectl — ArgoCD checks Git every 3 minutes by default
# you can also click "Sync" in the ArgoCD UI
```

### Check what ArgoCD is doing

```bash
kubectl get applications -n argocd          # list all ArgoCD apps
kubectl describe application grafana -n argocd   # details + sync status
```

### Restart k3s (if needed)

```bash
sudo systemctl restart k3s      # restart
sudo systemctl status k3s       # check status
sudo systemctl enable k3s       # make it start on boot (already set)
```

---

## 10. Troubleshooting Cheatsheet

### Pod is not starting — check why

```bash
kubectl describe pod <pod-name> -n <namespace>   # look at Events section at bottom
kubectl logs <pod-name> -n <namespace>           # app logs
kubectl logs <pod-name> -n <namespace> --previous   # logs before crash
```

### Pod is in CrashLoopBackOff

The container keeps crashing. Read the logs:
```bash
kubectl logs <pod-name> --previous
```

### Pod is in Pending

Usually means no node has enough resources, or a PVC is not being provisioned:
```bash
kubectl describe pod <pod-name>   # Events will say why it's pending
kubectl get pvc -A                # check if any PVC is Pending
```

### Ingress not working (can't reach a domain)

1. Check the Ingress exists: `kubectl get ingress -A`
2. Check Traefik is running: `kubectl get pods -n kube-system`
3. Check the Service the Ingress points to exists: `kubectl get svc -n <namespace>`
4. Check the certificate: `kubectl get certificate -A`

### ArgoCD app stuck in sync

```bash
kubectl describe application <name> -n argocd   # see error message
# Common causes: wrong repoURL, wrong chart version, missing namespace
```

### See resource usage

```bash
kubectl top nodes
kubectl top pods -A
```

---

## 11. Glossary

| Term | What it means |
|---|---|
| **Pod** | One running instance of your app (wraps containers) |
| **Deployment** | Manages multiple pod replicas, handles rollouts |
| **Service** | Stable network endpoint for reaching pods |
| **Ingress** | HTTP/HTTPS routing rules (Traefik reads these) |
| **Namespace** | A virtual partition of the cluster |
| **ConfigMap** | Non-secret configuration data |
| **Secret** | Sensitive data (passwords, certs) |
| **PVC** | Persistent Volume Claim — request for durable storage |
| **Helm** | Package manager for Kubernetes (like apt/npm but for k8s apps) |
| **Kustomize** | Tool to customize YAML without forking it |
| **ArgoCD** | GitOps tool — watches Git and keeps cluster in sync |
| **Traefik** | Ingress controller — routes external traffic into cluster |
| **cert-manager** | Automatically issues and renews TLS certificates |
| **ClusterIssuer** | cert-manager config for how to get certificates (Let's Encrypt) |
| **sourceoftruth** | Your root ArgoCD app — it watches `apps/` and deploys everything |
| **app of apps** | Pattern where one ArgoCD app manages all other ArgoCD apps |
| **k3s** | Lightweight single-binary Kubernetes |
| **containerd** | The container runtime k3s uses (not Docker) |
| **KUBECONFIG** | File that tells kubectl how to connect to your cluster (`~/.kube/config`) |

---

*Last updated: May 2026 — cluster running k3s v1.34.4, kubectl v1.35.3, Docker 29.3.1*
