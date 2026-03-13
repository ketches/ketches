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
  --set config.dbHost=my-postgres \
  --set config.dbPort=5432 \
  --set config.dbName=ketches \
  --set config.dbUsername=ketches \
  --set config.dbPassword=secret \
  --set config.dbSSLMode=disable
  --set config.dbAutoMigrate=true
```

If you prefer, you can still set `config.dbSource` directly.

**Note**: `config.dbAutoMigrate` is available. For production environments, set `--set config.dbAutoMigrate=false` and run schema migrations separately.

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
