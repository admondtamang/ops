# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A GitOps configuration repository for a single-node k3s cluster running at `https://127.0.0.1:6443`. Everything in this repo is declarative YAML — there is no application code, no build system, and no test suite. Changes are deployed by pushing to GitHub; ArgoCD watches the repo and syncs to the cluster automatically.

## Key commands

```bash
# Bootstrap ArgoCD (Helm install, run from bootstrap/argocd/)
cd bootstrap/argocd && ./install.sh

# Apply cert-manager + ClusterIssuers (from repo root)
kubectl apply -k ./kustomize

# Apply root ArgoCD app + ArgoCD ingress (from repo root)
kubectl apply -f argo-config.yaml

# Watch all pods across the cluster
kubectl get pods -A -w

# Check ArgoCD app sync status
kubectl get applications -n argocd

# Get initial ArgoCD admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo

# Force an ArgoCD sync (if CLI installed)
argocd app sync sourceoftruth
```

## Architecture

### GitOps flow

```
Push to GitHub (admondtamang/ops)
  → ArgoCD polls every 180s (timeout.reconciliation)
    → sourceoftruth app watches apps/ directory
      → Each YAML in apps/ is an ArgoCD Application
        → ArgoCD deploys/syncs each app to the cluster
```

`argo-config.yaml` defines two things: the `sourceoftruth` root application (which points ArgoCD at the `apps/` directory) and the ArgoCD ingress at `argo.admondtamang.com.np`.

### Deployment order

When rebuilding from scratch, the order matters:
1. `./install.sh` — installs ArgoCD via Helm
2. `kubectl apply -k ./kustomize` — installs cert-manager + `letsencrypt-prod` ClusterIssuer
3. `kubectl apply -f argo-config.yaml` — activates GitOps; ArgoCD takes over from here

### TLS

All HTTPS certificates use Let's Encrypt via cert-manager. Add the annotation `cert-manager.io/cluster-issuer: letsencrypt-prod` to any Ingress and cert-manager handles issuance automatically. The HTTP-01 challenge goes through Traefik (the k3s built-in ingress controller). Domain: `*.admondtamang.com.np`.

### Adding a new app

Copy an existing file from `apps/` (e.g., `grafana.yaml`) as a template. The pattern is:
- `spec.sources[].chart` + `repoURL` + `targetRevision` — Helm chart reference
- `spec.sources[].helm.values` — inline Helm values (include ingress + TLS annotations)
- `spec.destination.namespace` — usually `default`
- `syncPolicy.automated.prune: true` + `selfHeal: true` — standard for all apps

Push the file; ArgoCD will create the Application automatically (no `kubectl apply` needed once `sourceoftruth` is running).

### graveyard/

Old configs removed from active use. Kept for reference only — do not re-apply without review.

## Cluster facts

- k3s v1.34, single node named `zmond`
- Ingress controller: Traefik (built into k3s), node IP `192.168.1.64`
- Storage: `local-path-provisioner` (automatic PVC provisioning on node disk)
- ArgoCD reconciliation interval: 180s
- ArgoCD RBAC: default role is `readonly`; `role:org-admin` has full app/repo/cluster access
- ArgoCD HA and Dex (SSO) are disabled to conserve resources on the single node
