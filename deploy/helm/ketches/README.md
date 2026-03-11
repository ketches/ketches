# Ketches Helm Chart

This chart deploys:

- `ketches-api` (Go backend)
- `ketches-ui` (Nginx-hosted frontend)
- Optional bundled PostgreSQL (`postgres.enabled=true` by default)

## Prerequisites

- Kubernetes 1.24+
- Helm 3.12+

## Install

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --create-namespace
```

## Common Overrides

### Use external database

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --create-namespace \
  --set postgres.enabled=false \
  --set config.dbDriver=postgres \
  --set-string config.dbSource='host=my-postgres port=5432 user=ketches password=secret dbname=ketches sslmode=disable'
```

### Update images

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --set api.image.tag=v1.0.0 \
  --set ui.image.tag=v1.0.0
```

### Enable ingress

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=ketches.example.com
```

## Access

By default UI service type is `NodePort` (`30080`):

```bash
kubectl -n ketches get svc ketches-ui
```

If you switch UI service to `ClusterIP`, use port-forward:

```bash
kubectl -n ketches port-forward svc/ketches-ui 8080:80
```
