# Spec 05 — Git State vs Cluster State

## The core mental model

ArgoCD is a reconciliation loop. Every 180 seconds (your config) it:

1. Reads your Git repo
2. Reads the current cluster state
3. Compares them
4. Acts on the difference (if sync policy allows)

Understanding what's in Git vs what's in the cluster is the most important
skill for debugging ArgoCD problems.

## The three states an Application can be in

```
Synced    — Git matches cluster. ArgoCD is happy.
OutOfSync — Git differs from cluster. ArgoCD wants to act.
Unknown   — ArgoCD can't tell (usually a permissions or connection issue).
```

And separately, the health:

```
Healthy   — All resources are running / ready
Degraded  — Something is failing (crash loop, failed pod, etc.)
Progressing — Resources are being created/updated
Missing   — ArgoCD expects a resource that doesn't exist yet
```

You can see both at once:
```bash
kubectl get applications -n argocd
# NAME        SYNC STATUS   HEALTH STATUS
# bootstrap   Synced        Healthy
# nats        Synced        Healthy
# myapp       OutOfSync     Progressing
```

## What "Synced" actually means

**Synced does NOT mean "the pods are running fine."**
It means "the YAML I applied matches what's in Git."

Example: you push a typo in a Deployment image name.
- ArgoCD applies it → status becomes Synced (Git matches cluster ✓)
- Pods fail to pull the image → status becomes Degraded
- ArgoCD is Synced AND Degraded at the same time

This is why the review caught the case where `argo-config.yaml` was "Synced"
but the `delete_request_store` fix wasn't in Git yet — ArgoCD had synced
the old ConfigMap correctly.

## What triggers a sync

| Event | ArgoCD reaction |
|---|---|
| Git push | ArgoCD polls every 180s, picks it up |
| `argocd app sync <name>` | Immediate sync |
| Hard refresh in UI | Clears cache, re-fetches Git |
| `selfHeal: true` + cluster drift | Immediate re-sync |

## The out-of-band trap

"Out-of-band" means a change you made directly to the cluster with `kubectl`
that was never in Git.

```bash
kubectl edit configmap loki-config -n monitoring
# You add delete_request_store directly to the cluster

kubectl get applications -n argocd
# loki    Synced    Healthy   ← still shows Synced!
```

Why? Because ArgoCD compared Git (no delete_request_store) against the cluster
(has delete_request_store). Git won — it applied the Git version and removed
your change. The ConfigMap shows Synced because it now matches Git again.

**The fix must go into Git, not directly into the cluster.**
This is the rule that makes GitOps work.

## Reading sync diff

When an app is OutOfSync, you can see exactly what differs:

```bash
argocd app diff <app-name>

# Or via kubectl:
kubectl describe application myapp -n argocd | grep -A 20 "Sync Status"
```

The diff shows Git (desired) vs cluster (live). Green lines are what Git wants
to add. Red lines are what exists in the cluster but not in Git.

## Forcing a sync when ArgoCD is stuck

```bash
# Normal sync (respects prune/selfHeal settings)
argocd app sync <app-name>

# Hard refresh (clears ArgoCD's cached state, re-fetches from Git)
argocd app get <app-name> --hard-refresh

# Sync with prune (explicitly deletes resources not in Git)
argocd app sync <app-name> --prune
```

Hard refresh is the first thing to try when "it should be synced but isn't" —
ArgoCD caches the rendered Helm output and can serve stale values.

## The finalizer question

From `apps/myapp.yaml`:
```yaml
metadata:
  finalizers:
    - resources-finalizer.argocd.argoproj.io
```

This finalizer means: **when you delete the Application object, also delete
everything it manages** (Deployments, Services, Pods, etc.).

Without it: deleting the Application object orphans the resources.
The pods keep running with no Application watching them — they become invisible
to ArgoCD but still consume cluster resources.

Your `aoa.yaml` (root Application) has **no finalizer**. That's intentional:
if you delete the root app, you want the child Applications to stay running
rather than cascade-deleting your entire cluster.

## Summary

| Concept | What to check |
|---|---|
| Is ArgoCD watching my app? | `kubectl get applications -n argocd` |
| Does Git match the cluster? | `argocd app diff <name>` |
| Why is a pod failing? | `kubectl describe pod <name>` — this is cluster state, not Git |
| Why isn't my Git push applying? | Check sync status; try hard refresh |
| Did my kubectl change stick? | Check if selfHeal is true — it may have been reverted |
