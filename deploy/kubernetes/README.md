# Kubernetes Deployment Files

This directory provides two raw Kubernetes manifest paths.

## 1. Quickstart manifest

Use `manifests.quickstart.yaml` only for local evaluation:

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/ketches/master/deploy/kubernetes/manifests.quickstart.yaml
```

This file intentionally includes:

- demo secrets
- localhost-friendly `CORS_ALLOWED_ORIGINS`
- `DB_SSLMODE=disable` for local-only database access

Do not reuse it in shared, staging, or production environments.

## 2. Secure-by-default manifest

Use `manifests.yaml` for real deployments:

```bash
kubectl apply -f deploy/kubernetes/manifests.yaml
```

This file keeps secret fields blank on purpose so operators must provide:

- database password
- JWT signing secret
- secret-encryption key
- real CORS origins

## Which one should I use?

- Local demo / evaluation cluster → `manifests.quickstart.yaml`
- Shared environment / production / anything internet-facing → `manifests.yaml`

## Next step for production

Before using `manifests.yaml` outside local evaluation, read [Production Deployment Guide](../../docs/PRODUCTION_DEPLOYMENT.md).
