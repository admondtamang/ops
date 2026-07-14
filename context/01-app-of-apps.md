# Spec 01 — App-of-Apps Pattern

## The problem it solves

ArgoCD needs to know about each application you want it to manage.
You could run `kubectl apply -f apps/nats.yaml` for every app manually —
but then you're back to doing things by hand and GitOps is pointless.

The app-of-apps pattern solves this: you tell ArgoCD about **one** special
Application that watches a folder. ArgoCD reads that folder and creates all
the other Applications automatically.

## Your setup

```
aoa.yaml  (you apply this once, manually)
  └── watches apps/ directory
        ├── apps/nats.yaml       → ArgoCD creates "nats" Application
        ├── apps/myapp.yaml      → ArgoCD creates "myapp" Application
        ├── apps/vault.yaml      → ArgoCD creates "vault" Application
        └── apps/...             → and so on
```

The root Application in `aoa.yaml`:

```yaml
metadata:
  name: bootstrap          # the root — manages everything else
spec:
  source:
    path: apps             # watches this directory
    directory:
      recurse: true        # picks up subdirectories too (e.g. apps/monitoring/)
```

Each file in `apps/` is itself an ArgoCD `Application` pointing at a Helm chart
or a directory of manifests. ArgoCD applies them all.

## What "managing" means

When ArgoCD "manages" an Application, it:
1. Reads the YAML from your repo
2. Applies it to the cluster
3. Keeps watching — if the cluster drifts from the repo, it re-syncs

## The critical rule

**The root Application (`bootstrap`) is outside `apps/`.** It cannot manage itself.
That's why you apply it manually with `kubectl apply -f aoa.yaml`.

If the root Application disappears from the cluster:
- ArgoCD stops watching `apps/`
- All child apps still run (pods don't stop immediately)
- But no new syncs happen — the cluster freezes at its last state
- Add a new file to `apps/` and nothing will happen until you re-apply `aoa.yaml`

## What just happened in your review

Your staged commit deleted the old root Application (`sourceoftruth`) from
`argo-config.yaml`. The replacement (`bootstrap` in `aoa.yaml`) was not staged.

Had you pushed: `aoa.yaml` would never reach GitHub. After any cluster rebuild
you'd apply `argo-config.yaml`, get only an Ingress, no root app, and
the cluster would be completely inert — with no error message to tell you why.

## Next file

`02-bootstrap-sequence.md` — why the order of `kubectl apply` commands matters
and what goes wrong if you get it wrong.
