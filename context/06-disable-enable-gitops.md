# Spec 06 — Disabling and Re-enabling GitOps

## Why you'd do this

When ArgoCD manages a resource with `selfHeal: true`, any `kubectl` change
you make gets reverted within seconds. To apply changes directly to the cluster
without ArgoCD fighting you, you delete the root Application temporarily.

Because `aoa.yaml` has no `resources-finalizer`, deleting the root app only
removes the watcher — all running workloads (pods, services, deployments) stay
up. Nothing restarts. Nothing breaks.

## Disable GitOps (stop ArgoCD from managing apps)

```bash
kubectl delete application bootstrap -n argocd
```

What happens:
- The `bootstrap` Application object is removed from the cluster
- ArgoCD stops watching `apps/` from Git
- All 13 child Applications still exist and keep running
- `selfHeal` and `prune` stop firing — you can now `kubectl` freely

Verify it's gone:
```bash
kubectl get applications -n argocd
# bootstrap should no longer appear
# all other apps still listed
```

## Re-enable GitOps (hand control back to ArgoCD)

```bash
cd ~/Work/ops && kubectl apply -f aoa.yaml
```

What happens:
- `bootstrap` Application is re-created in the cluster
- ArgoCD re-reads `apps/` from Git
- Any drift you introduced manually will be reverted within 180s (or immediately
  if selfHeal is true on the child apps)

Verify it's back:
```bash
kubectl get applications -n argocd
# bootstrap should appear, status will move from Unknown → Synced
```

## Important: what ArgoCD will revert when you re-enable

When GitOps comes back, ArgoCD compares Git against the cluster.
Any manual change you made that differs from Git will be reverted.

If you want your manual change to survive:
1. Make the change in Git first, then re-enable GitOps
2. Or: push your change to Git after making it manually, then re-enable

If you re-enable GitOps without committing your changes:
- ArgoCD sees drift → reverts to Git state → your work is undone

## The no-finalizer rule

`aoa.yaml` intentionally has no `resources-finalizer.argocd.argoproj.io`.

```yaml
metadata:
  name: bootstrap
  namespace: argocd
  # no finalizers — intentional
```

Without the finalizer: deleting `bootstrap` is safe. Only the watcher is removed.

With the finalizer: deleting `bootstrap` would cascade-delete all child
Applications, which would cascade-delete all pods across the cluster.
That would take down every workload. Never add the finalizer to the root app.

Compare: child apps like `myapp` DO have the finalizer because you want
`kubectl delete application myapp` to fully clean up NATS pods, services, etc.
The root app is the exception.

## Quick reference

| Goal | Command |
|---|---|
| Stop ArgoCD managing everything | `kubectl delete application bootstrap -n argocd` |
| Resume ArgoCD | `cd ~/Work/ops && kubectl apply -f aoa.yaml` |
| Check what's still running | `kubectl get applications -n argocd` |
| Check if pods survived | `kubectl get pods -A` |
