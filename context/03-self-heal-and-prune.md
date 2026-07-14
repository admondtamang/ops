# Spec 03 — selfHeal and prune

These are two separate flags that control what ArgoCD does when Git and the
cluster disagree. They look similar but they guard against opposite scenarios.

## prune — handles things that exist in cluster but not in Git

**Without prune:** you delete `apps/nats.yaml` from Git. ArgoCD syncs.
The NATS pods keep running forever — ArgoCD just stops watching them.

**With prune: true:** you delete `apps/nats.yaml` from Git. ArgoCD syncs.
ArgoCD deletes the NATS Application, which (if the nats Application itself has
`prune: true`) cascades down and removes the pods too.

```
Git change: file deleted
prune: false → cluster resource stays (orphaned)
prune: true  → cluster resource is deleted
```

Your root app (`aoa.yaml`) and all apps in `apps/` have `prune: true`.
This is correct for a GitOps cluster — Git is the source of truth,
so things that leave Git should leave the cluster.

## selfHeal — handles things that exist in cluster but differ from Git

**Without selfHeal:** you run `kubectl edit deployment myapp` and change
replica count to 0 to stop it temporarily. ArgoCD notices the drift but
does nothing — it waits for the next sync trigger (e.g. a new Git push).

**With selfHeal: true:** you run `kubectl edit deployment myapp`. Within
seconds ArgoCD reverts your change back to what Git says. You cannot make
a manual change stick.

```
Cluster change: manual kubectl edit
selfHeal: false → drift is tolerated until next sync
selfHeal: true  → drift is reverted immediately (~seconds)
```

## Your current situation

Your child apps (`myapp`, `nats`, etc.) already have `selfHeal: true`.
The old root app (`sourceoftruth`) had `selfHeal: false`.
Your new root app (`aoa.yaml`) has `selfHeal: true`.

The root app manages `Application` objects in the `apps/` directory.
With `selfHeal: true` on the root: if you `kubectl edit application nats`
to temporarily change a value for debugging, ArgoCD reverts it.

## When selfHeal: true causes problems

```bash
# You want to temporarily disable vault to free memory
kubectl patch application vault -n argocd \
  --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'

# With selfHeal: true on bootstrap — ArgoCD reverts this in ~10 seconds.
# You're now in a fight with ArgoCD.
```

The break-glass procedure when selfHeal fights you:
```bash
# Pause the ArgoCD controller temporarily
kubectl scale deployment argocd-application-controller -n argocd --replicas=0

# Make your manual change
kubectl edit ...

# Resume
kubectl scale deployment argocd-application-controller -n argocd --replicas=1
```

## Summary table

| Flag | Triggers when | Effect |
|---|---|---|
| `prune: true` | File removed from Git | Deletes cluster resource |
| `prune: false` | File removed from Git | Cluster resource stays |
| `selfHeal: true` | Cluster drifts from Git | ArgoCD reverts immediately |
| `selfHeal: false` | Cluster drifts from Git | ArgoCD waits, does nothing |

## The rule of thumb

- `prune: true` — almost always what you want on a GitOps cluster
- `selfHeal: true` — good for production discipline, inconvenient for active debugging
- On a learning cluster: `selfHeal: false` gives you room to experiment manually

## Next file

`04-ingress-tls.md` — why the TLS host and the rules host must match,
and what happens when they don't.
