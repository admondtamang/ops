# Spec 04 — Ingress and TLS

## What an Ingress does

An Ingress is a routing rule. It says:
"When a request arrives at the cluster for host X and path Y, send it to service Z."

Traefik (your ingress controller, built into k3s) reads all Ingress objects
and acts as the traffic cop.

## The three parts of a TLS Ingress

```yaml
spec:
  tls:                              # Part 1 — which certs to serve
    - hosts:
        - argo.admondtamang.com.np  # serve this cert for this hostname
      secretName: argo-tls          # cert stored in this Secret
  rules:                            # Part 2 — where to route traffic
    - host: argo.admondtamang.com.np  # requests for THIS hostname...
      http:
        paths:
          - path: /
            backend:
              service:
                name: argocd-server  # ...go to this service
                port:
                  number: 80
```

**The hostname in `tls.hosts` and `rules.host` must match.**

Traefik uses `tls.hosts` to decide which TLS certificate to present for
an incoming request. It uses `rules.host` to decide where to route the request.
If they don't match, the connection gets TLS (or doesn't) based on one hostname
but routing based on another — the request falls through or gets a 404.

## Your current mismatch

After your staged change, `argo-config.yaml` looks like:

```yaml
spec:
  tls:
    - hosts:
        - argo.admondtamang.com.np   # cert covers the PUBLIC domain
      secretName: argo-tls
  rules:
    - host: local.argocd             # route requests for LOCAL hostname
```

These are two different hostnames. What this produces:

- A request to `https://argo.admondtamang.com.np` → gets the TLS cert but
  no matching rule → 404 from Traefik
- A request to `http://local.argocd` (if you add it to `/etc/hosts`) →
  matches the rule but gets no TLS → ArgoCD tokens and auth over plain HTTP

## The fix

Pick one consistent hostname. For a cluster with a real domain:

```yaml
spec:
  tls:
    - hosts:
        - argo.admondtamang.com.np
      secretName: argo-tls
  rules:
    - host: argo.admondtamang.com.np   # matches tls.hosts exactly
```

If you also want a local alias, add a second rule:

```yaml
  rules:
    - host: argo.admondtamang.com.np   # public, TLS
      http: ...
    - host: local.argocd               # local, no TLS (fine for /etc/hosts access)
      http: ...
```

## How cert-manager fits in

cert-manager reads Ingress objects looking for this annotation:

```yaml
annotations:
  cert-manager.io/cluster-issuer: letsencrypt-prod
```

When it sees that annotation, it:
1. Reads `tls[].hosts` to know what domain to request a cert for
2. Sends an HTTP-01 challenge through Traefik to Let's Encrypt
3. Stores the resulting cert in `tls[].secretName`

If the domain in `tls.hosts` doesn't match the Ingress rule host, cert-manager
still issues the cert — but Traefik can't serve it because no request ever
matches both the cert domain and the routing rule simultaneously.

## Your NATS example (working correctly)

From `apps/nats.yaml`:
```yaml
websocket:
  ingress:
    hosts:
      - nats.admondtamang.com.np        # rule host
    tlsSecretName: nats-tls
    merge:
      metadata:
        annotations:
          cert-manager.io/cluster-issuer: letsencrypt-prod
```

The NATS chart generates an Ingress where the rule host and the TLS host
both use `nats.admondtamang.com.np`. This is correct.

## Next file

`05-git-vs-cluster-state.md` — the mental model of what ArgoCD does when
Git and the cluster disagree, and how to read the ArgoCD sync status.
