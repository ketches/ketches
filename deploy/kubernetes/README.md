# Kubernetes Deployment Files

This directory provides two raw Kubernetes manifest paths.

## 1. Quickstart manifest

Use `manifests.quickstart.yaml` only for local evaluation. Create the Secret first; the manifest intentionally does not contain bootstrap credentials:

```bash
kubectl create namespace ketches --dry-run=client -o yaml | kubectl apply -f -
BOOTSTRAP_ADMIN_PASSWORD="$(openssl rand -base64 24 | tr -d '\n')"
kubectl -n ketches create secret generic ketches-secrets \
  --from-literal=db-password=ketches-quickstart-postgres \
  --from-literal=jwt-secret=ketches-quickstart-jwt-secret \
  --from-literal=secret-encryption-key=ketches-quickstart-encryption-key \
  --from-literal=bootstrap-admin-username=kadmin \
  --from-literal=bootstrap-admin-password="$BOOTSTRAP_ADMIN_PASSWORD" \
  --from-literal=smtp-password= \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f https://raw.githubusercontent.com/ketches/ketches/master/deploy/kubernetes/manifests.quickstart.yaml
```

This file intentionally includes:

- localhost-friendly `CORS_ALLOWED_ORIGINS`
- `DB_SSLMODE=disable` for local-only database access

Do not reuse it in shared, staging, or production environments.

## 2. Secure-by-default manifest

Use `manifests.yaml` for real deployments:

```bash
kubectl create namespace ketches --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ketches create secret generic ketches-secrets \
  --from-literal=db-password='<db-password>' \
  --from-literal=jwt-secret='<jwt-secret>' \
  --from-literal=secret-encryption-key='<secret-encryption-key>' \
  --from-literal=bootstrap-admin-username='kadmin' \
  --from-literal=bootstrap-admin-password="$(openssl rand -base64 24 | tr -d '\n')" \
  --from-literal=smtp-password='' \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/kubernetes/manifests.yaml
```

This file does not create the Secret, so operators must provide:

- database password
- JWT signing secret
- secret-encryption key
- random bootstrap administrator password
- real CORS origins

## Which one should I use?

- Local demo / evaluation cluster → `manifests.quickstart.yaml`
- Shared environment / production / anything internet-facing → `manifests.yaml`

## Next step for production

Before using `manifests.yaml` outside local evaluation, read [Production Deployment Guide](../../docs/PRODUCTION_DEPLOYMENT.md).
