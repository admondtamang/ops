# CLAUDE.md

GitOps repo for single-node k3s @ `https://127.0.0.1:6443`. Pure declarative YAML. Push to GitHub → ArgoCD syncs.

## Commands
```bash
cd bootstrap/argocd && ./install.sh          # bootstrap ArgoCD
kubectl apply -k ./kustomize                 # cert-manager + ClusterIssuer
kubectl apply -f argo-config.yaml            # root app + ingress (then ArgoCD owns it)
kubectl get pods -A -w                       # watch pods
kubectl get applications -n argocd           # sync status
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
argocd app sync sourceoftruth                # force sync
```

## Architecture
`argo-config.yaml` → `sourceoftruth` app watches `apps/` → each YAML = ArgoCD Application → cluster.
Poll interval: 180s. Ingress: `argo.admondtamang.com.np`.

**Bootstrap order (scratch):** install.sh → kustomize → argo-config.yaml

**TLS:** cert-manager + Let's Encrypt HTTP-01 via Traefik. Annotation: `cert-manager.io/cluster-issuer: letsencrypt-prod`. Domain: `*.admondtamang.com.np`.

**New app:** copy `apps/grafana.yaml`. Required fields: `spec.sources[].{chart,repoURL,targetRevision}`, `helm.values` (inline, include ingress+TLS), `destination.namespace`, `syncPolicy.automated.{prune,selfHeal}: true`. Push → ArgoCD auto-creates.

**graveyard/:** dead configs, reference only, do not re-apply.

## Cluster
- k3s v1.34, node: `zmond`, IP: `192.168.1.64`
- Ingress: Traefik (built-in), Storage: `local-path-provisioner`
- ArgoCD RBAC: default=`readonly`, `role:org-admin`=full; HA+Dex disabled
