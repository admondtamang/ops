# Command Reference

Practical commands for managing this k3s + ArgoCD GitOps cluster.

## Table of Contents

- [ArgoCD](#argocd)
- [Pods](#pods)
- [Namespaces](#namespaces)
- [Services & Networking](#services--networking)
- [Helm (via ArgoCD)](#helm-via-argocd)
- [TLS / cert-manager](#tls--cert-manager)
- [Storage](#storage)
- [Debugging Scenarios](#debugging-scenarios)
- [GitOps Workflow](#gitops-workflow)

---

## ArgoCD

### Check all app statuses
```bash
kubectl get applications -n argocd
```
Shows every app's sync status (Synced/OutOfSync) and health (Healthy/Degraded/Progressing).

### Check a specific app
```bash
kubectl get application uptime-kuma -n argocd -o jsonpath='{.status.sync.status} {.status.health.status}'
```

### Force a sync (when you don't want to wait 180s)
```bash
kubectl annotate application <app-name> -n argocd argocd.argoproj.io/refresh=hard --overwrite
```
> **Scenario:** You pushed to GitHub and don't want to wait for the next poll cycle.

### Sync from local files without pushing to GitHub (requires argocd CLI)
```bash
argocd login argo.admondtamang.com.np
argocd app sync <app-name> --local ./apps
```
> **Scenario:** You're iterating on a YAML locally and want to test it immediately.

### See what resources an app manages
```bash
kubectl get application <app-name> -n argocd \
  -o jsonpath='{range .status.resources[*]}{.kind}/{.name} ns={.namespace}{"\n"}{end}'
```

### Get ArgoCD admin password
```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo
```

---

## Pods

### Watch all pods across the cluster
```bash
kubectl get pods -A -w
```
> **Scenario:** You just pushed a change — watch it roll out in real time.

### Find a pod by partial name
```bash
kubectl get pods -A | grep kuma
```

### Check why a pod is crashing
```bash
kubectl describe pod <pod-name> -n <namespace>
```
Scroll to the `Events:` section at the bottom — that's where the error is.

### Read pod logs
```bash
kubectl logs <pod-name> -n <namespace>
```

### Follow logs in real time
```bash
kubectl logs -f <pod-name> -n <namespace>
```
> **Scenario:** You deployed n8n and want to watch it start up live.

### Read logs from a crashed pod (previous run)
```bash
kubectl logs <pod-name> -n <namespace> --previous
```
> **Scenario:** Pod keeps restarting and you can't catch the logs before it dies.

### Shell into a running pod
```bash
kubectl exec -it <pod-name> -n <namespace> -- /bin/sh
```
Use `/bin/bash` if sh isn't available.

---

## Namespaces

### List all namespaces
```bash
kubectl get namespaces
```

### See everything running in a namespace
```bash
kubectl get all -n <namespace>
```

### Create a namespace manually
```bash
kubectl create namespace <name>
```
> Note: With `CreateNamespace=true` in ArgoCD syncOptions, you rarely need this.

### Delete a namespace
```bash
kubectl delete namespace <name>
```

### Fix a namespace stuck in Terminating
```bash
kubectl get namespace <name> -o json \
  | python3 -c "import sys,json; ns=json.load(sys.stdin); ns['spec']['finalizers']=[]; print(json.dumps(ns))" \
  | kubectl replace --raw /api/v1/namespaces/<name>/finalize -f -
```
> **Scenario:** You deleted a namespace but it hangs forever in `Terminating` state.

---

## Services & Networking

### List all services
```bash
kubectl get svc -A
```

### Port-forward a service to your local machine
```bash
kubectl port-forward svc/<service-name> <local-port>:<service-port> -n <namespace>
```
Example — access Grafana on localhost:3000:
```bash
kubectl port-forward svc/grafana 3000:80 -n default
```
> **Scenario:** You want to access an app that has no ingress configured yet.

### Port-forward a deployment directly
```bash
kubectl port-forward deployment/<name> <local-port>:<container-port> -n <namespace>
```

### Check ingress rules
```bash
kubectl get ingress -A
```

---

## Helm (via ArgoCD)

ArgoCD manages Helm for you — you rarely run `helm` directly. But if you need to debug a chart's rendered output:

### Preview what a Helm chart will generate (dry run)
```bash
helm template <release-name> <chart> \
  --repo <repo-url> \
  --version <version> \
  -f values.yaml
```
> **Scenario:** You're not sure if your Helm values are correct before committing.

---

## TLS / cert-manager

### Check certificate status
```bash
kubectl get certificates -A
```
`READY=True` means the cert was issued successfully.

### See why a cert isn't issuing
```bash
kubectl describe certificate <cert-name> -n <namespace>
kubectl describe certificaterequest -n <namespace>
```
> **Scenario:** Your app's HTTPS shows a certificate error in the browser.

### Check ClusterIssuers (Let's Encrypt config)
```bash
kubectl get clusterissuers
```

---

## Storage

### List all persistent volume claims
```bash
kubectl get pvc -A
```

### Check if a PVC is bound (has storage allocated)
```bash
kubectl get pvc -n <namespace>
```
`STATUS=Bound` means the pod has its storage. `Pending` means it's waiting — check events.

---

## Debugging Scenarios

### App is OutOfSync in ArgoCD
```bash
# See what ArgoCD wants to change
kubectl get application <app-name> -n argocd -o jsonpath='{.status.sync.comparedTo}'
# Force a sync
kubectl annotate application <app-name> -n argocd argocd.argoproj.io/refresh=hard --overwrite
```

### Pod is in CrashLoopBackOff
```bash
kubectl describe pod <pod-name> -n <namespace>   # check Events section
kubectl logs <pod-name> -n <namespace> --previous # read last crash output
```

### Pod is Pending (never starts)
```bash
kubectl describe pod <pod-name> -n <namespace>
# Common causes shown in Events:
#   "Insufficient memory" — node is out of RAM
#   "did not find available node" — scheduling failed
#   "PVC not bound" — storage isn't provisioned
```

### Ingress works but certificate shows as untrusted
```bash
kubectl get certificate -n <namespace>
kubectl describe certificate <name> -n <namespace>
# Check the challenge:
kubectl get challenges -A
```
> Usually means the HTTP-01 challenge failed. Check that your domain DNS points to `192.168.1.64` and port 80 is reachable.

### Node resource usage
```bash
kubectl top nodes
kubectl top pods -A
```
> Requires metrics-server. If it errors, metrics-server may not be running.

---

## GitOps Workflow

### The normal deploy flow
```bash
# 1. Edit a YAML in apps/
vim apps/myapp.yaml

# 2. Commit and push
git add apps/myapp.yaml
git commit -m "fix: update myapp config"
git push

# 3. ArgoCD auto-syncs within 180s — or force it:
kubectl annotate application myapp -n argocd argocd.argoproj.io/refresh=hard --overwrite

# 4. Watch the rollout
kubectl get pods -n <namespace> -w
```

### Add a new app
Copy an existing file from `apps/` and push it. ArgoCD's `sourceoftruth` app watches the `apps/` directory and will create the new Application automatically — no `kubectl apply` needed.

### Remove an app
Delete the file from `apps/` and push. With `prune: true`, ArgoCD will delete all resources that app created.
