# Spec 02 — Bootstrap Sequence

## The two worlds

Your cluster has two kinds of resources:

| World | How it gets there | Who watches it |
|---|---|---|
| **GitOps** | ArgoCD reads Git, applies to cluster | ArgoCD (automatic) |
| **Bootstrap** | You run `kubectl apply` manually | Nobody — it's static |

The bootstrap world exists only to start the GitOps world.
Once GitOps is running, you stop using `kubectl apply` for everything else.

## Your bootstrap files

```
bootstrap/argocd/install.sh          # installs ArgoCD itself via Helm
bootstrap/argocd/argo-config.yaml    # ArgoCD Ingress (static, manually applied)
aoa.yaml                             # root Application (manually applied once)
kustomize/                           # cert-manager + ClusterIssuer (manually applied)
```

## The correct order (from your CLAUDE.md)

```bash
# Step 1 — Install ArgoCD
cd bootstrap/argocd && ./install.sh

# Step 2 — Install cert-manager + TLS issuer
kubectl apply -k ./kustomize

# Step 3 — Apply the ArgoCD Ingress (so you can reach the UI)
kubectl apply -f argo-config.yaml

# Step 4 — Apply the root Application (hands control to GitOps)
kubectl apply -f aoa.yaml
```

After step 4, ArgoCD reads `apps/` from GitHub and creates everything else.
**You never run `kubectl apply` for individual apps again** — you push to Git instead.

## Why step 4 must come last

`aoa.yaml` references `git@github.com:admondtamang/ops.git`. If you apply it
before ArgoCD is running (step 1), there's no controller to act on it.
If you apply it before cert-manager (step 2), apps that need TLS will fail.

The dependency chain:
```
ArgoCD running → can process Application objects
cert-manager running → can issue TLS certs
Ingress applied → you can reach the UI
Root Application applied → GitOps takes over
```

## What happens during a rename (your current situation)

You're renaming `sourceoftruth` → `bootstrap`. These are two different
Kubernetes objects. The rename does not happen automatically.

**Safe sequence:**
```bash
# 1. Push the commit (both argo-config.yaml change AND aoa.yaml)
git push

# 2. Create the new root app
kubectl apply -f aoa.yaml
kubectl get applications -n argocd   # wait for "bootstrap" to appear + sync

# 3. Delete the old root app (child apps keep running, no cascade)
kubectl delete application sourceoftruth -n argocd
```

The old `sourceoftruth` object stays in the cluster until you delete it manually
because it was applied out-of-band — ArgoCD doesn't manage it, so ArgoCD can't
remove it.

## The trap

`kubectl apply -f argo-config.yaml` used to do two things (Ingress + root app).
After your change it does only one (Ingress). The command still exits 0.
No error. Anyone rebuilding from scratch following old docs would get a
working ArgoCD UI but a completely inert cluster.

**Always update CLAUDE.md when the bootstrap sequence changes.**

## Next file

`03-self-heal-and-prune.md` — what selfHeal and prune actually do, and why
flipping selfHeal from false to true is not a rename.
