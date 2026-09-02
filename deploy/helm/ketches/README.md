# Ketches Helm Chart

This chart deploys:

- `ketches-api` (Go backend)
- `ketches-ui` (Nginx-hosted frontend)
- Optional bundled PostgreSQL (`postgres.enabled=true` by default)

## Prerequisites

- Kubernetes 1.24+
- Helm 3.12+

## Install

### Quickstart (local evaluation only)

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --create-namespace \
  -f ./deploy/helm/ketches/values-quickstart.yaml
```

This quickstart file is intentionally for local evaluation only. It includes demo secrets so you can try the chart immediately.
It also disables sign-up email verification by default while keeping bootstrap admin creation enabled.

On the first install, the chart generates a random bootstrap administrator password and stores it in the chart Secret. Upgrades reuse the existing Secret value. Retrieve it with:

```bash
kubectl get secret ketches-secrets -n ketches \
  -o jsonpath='{.data.bootstrap-admin-password}' | base64 -d; echo
```

The initial administrator must change this password before using other API operations.

### Production / shared environments

The chart requires `config.jwtSecret` and `config.secretEncryptionKey` on every install or upgrade. When the bundled PostgreSQL instance is enabled, set `postgres.auth.password` as well.

Relevant auth-related values:

```txt
config.bootstrapAdminUsername
config.bootstrapAdminPassword
config.signUpEmailVerificationRequired
config.smtpHost
config.smtpPort
config.smtpUsername
config.smtpPassword
config.smtpFrom
```

`config.bootstrapAdminPassword` is optional. When omitted, the chart generates it on first install and persists it in the Secret. Set it explicitly only when the value is already managed by an external secret workflow.

## Common Overrides

### Use external database

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --create-namespace \
  --set postgres.enabled=false \
  --set config.jwtSecret="$JWT_SECRET" \
  --set config.secretEncryptionKey="$SECRET_ENCRYPTION_KEY" \
  --set config.dbDriver=postgres \
  --set config.dbHost=my-postgres \
  --set config.dbPort=5432 \
  --set config.dbName=ketches \
  --set config.dbUsername=ketches \
  --set config.dbPassword="$DB_PASSWORD" \
  --set config.dbSSLMode=verify-full
```

If you prefer, you can still set `config.dbSource` directly.

The API applies the entity schema at startup by default. Set `config.dbAutoMigrate=false` when schema changes are managed by a separate migration process; the API will then avoid migration DDL.

The bundled `postgres:17-alpine` image uses group ID `70`. If you replace it with a PostgreSQL image that runs the server under a different group, set `postgres.tls.serverGroupID` to that group so PostgreSQL can read the generated private key.

The chart explicitly trusts private-cluster proxy ranges so the bundled UI proxy preserves per-client authentication rate limits. Set `config.trustedProxies` to the narrower CIDRs used by your ingress and UI pods when your cluster network is known.

### Update images

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --set api.image.tag=v1.0.0 \
  --set ui.image.tag=v1.0.0
```

### Configure build log persistence

Build log archives are stored on the API pod filesystem and require persistent storage if you want them to survive pod recreation.

```bash
helm upgrade --install ketches ./deploy/helm/ketches \
  --namespace ketches \
  --set config.buildLogBaseDir=/app/data/build-logs \
  --set config.buildLogRetentionDays=15 \
  --set api.persistence.enabled=true \
  --set api.persistence.size=10Gi
```

This first version assumes `api.replicaCount=1`. If you scale the API horizontally, move build log storage to shared storage or object storage before relying on archived logs.

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
kubectl -n ketches port-forward svc/ketches-ui 3000:3000
```
