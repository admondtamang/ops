# Go server → Docker → ArgoCD: full deploy flow

End-to-end example of shipping a Go HTTP server to the k3s cluster via GitOps.

## 1. The server (`main.go`)

Simple HTTP server on port 8080. No frameworks, no dependencies beyond stdlib.

## 2. Build the Docker image

The `Dockerfile` uses a two-stage build: `golang:1.26-alpine` to compile,
`alpine:3.19` to run. Keep the Go version in sync with `go.mod`.

```bash
docker build -t admondtamang/myapp:v1 .
docker push admondtamang/myapp:v1
```

To release a new version, bump the tag (`v2`, `v3`, …), push the image, then
update the image tag in `manifests/myapp/deployment.yml` and push to git.
ArgoCD will roll it out automatically.

## 3. Kubernetes manifests (`manifests/myapp/deployment.yml`)

Three resources in one file:

- **Deployment** — runs one replica of the container, rolling-update strategy
- **Service** — ClusterIP on port 80 → container port 8080
- **Ingress** — Traefik ingress at `myapp.admondtamang.com.np` with TLS via
  cert-manager (`letsencrypt-prod`)

## 4. ArgoCD Application (`apps/myapp.yaml`)

Points ArgoCD at `manifests/myapp/` in this repo. Once pushed, ArgoCD
reconciles every 180s and self-heals any drift.

## 5. Deploy

```bash
# First time — build and push the image, then:
git add apps/myapp.yaml manifests/myapp/
git commit -m "feat: add myapp"
git push
# ArgoCD picks it up within 180s. No kubectl apply needed.

# Update the app — change code, bump image tag in deployment.yml, then:
docker build -t admondtamang/myapp:v2 .
docker push admondtamang/myapp:v2
# edit manifests/myapp/deployment.yml: image: admondtamang/myapp:v2
git commit -am "feat: bump myapp to v2"
git push
```

## File map

```
exmaples/goserver/          ← this example (source + Dockerfile)
  main.go
  Dockerfile
  go.mod

manifests/myapp/            ← k8s manifests (deployed by ArgoCD)
  deployment.yml

apps/myapp.yaml             ← ArgoCD Application definition
```
