# Deployment using argocd

Applications:

- argocd
- nginx server using deployments
- nginx using helm
- uptime kuma

## Kustomize

kubectl apply -k ./kustomize


# Frequently used commands

```
kubectl port-forward -n cert-manager deployment/uptime-kuma 3001:3001
```