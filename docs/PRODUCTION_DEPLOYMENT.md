# Production Deployment Guide

This guide describes the minimum configuration expected when deploying Ketches to shared, staging, or production environments.

Do not reuse the quickstart demo values from:

- `deploy/docker/.env.quickstart`
- `deploy/helm/ketches/values-quickstart.yaml`

Those files exist only to make local evaluation easier.

## Required secrets and settings

Before deploying Ketches outside a local evaluation environment, configure these values explicitly.

### Application secrets

- `JWT_SECRET`
  - Required for signing JWT tokens.
  - Use a strong random secret.
- `SECRET_ENCRYPTION_KEY`
  - Required for encrypting sensitive values at rest.
  - Use a strong random secret.

### Database credentials

Set real database credentials instead of relying on demo values.

- Docker Compose: `POSTGRES_PASSWORD`
- Helm bundled PostgreSQL: `postgres.auth.password`
- Helm external database: `config.dbPassword` or `config.dbSource`
- Raw manifests: secret-backed `db-password`

### CORS configuration

Set `CORS_ALLOWED_ORIGINS` to the real UI origin or origins that will access the API.

Examples:

- `https://app.example.com`
- `https://app.example.com,https://admin.example.com`

Do not leave a local-development origin list in place when exposing the system publicly.

## Recommended production settings

### Database migration policy

Set `DB_AUTO_MIGRATE=false` in controlled production environments and run schema changes through your normal rollout process.

### PostgreSQL SSL

Prefer `DB_SSLMODE=require` or a stricter verified SSL mode when the database connection leaves the local node or cluster.

If you started from `deploy/docker/.env.quickstart`, replace or remove its `DB_SSLMODE=disable` value before promoting that configuration anywhere beyond local evaluation.

### Persistence

Ensure persistent storage is configured for:

- the runtime database
- build log retention if you rely on archived build logs

### Ingress and DNS

Review:

- ingress hostname
- TLS certificates
- external UI/backend URLs

## Deployment-specific notes

### Docker Compose

The checked-in `deploy/docker/docker-compose.yml` intentionally requires secret values from the environment. Provide a real `.env` file or export the required variables before starting the stack.

At minimum:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `SECRET_ENCRYPTION_KEY`
- `CORS_ALLOWED_ORIGINS`

### Helm

The main chart is secure by default and does not ship with reusable production secrets.

For production or shared clusters:

- do not use `values-quickstart.yaml`
- provide real values for `config.jwtSecret` and `config.secretEncryptionKey`
- provide `postgres.auth.password` when using the bundled PostgreSQL instance
- set `postgres.enabled=false` and provide `config.dbSource` or `config.db*` values when using an external database

### Raw Kubernetes manifests

The checked-in `deploy/kubernetes/manifests.yaml` keeps secret fields blank on purpose. Fill those secrets before applying the manifests.

## Common pitfalls

### Install or startup fails immediately

Common cause: required secrets were left empty.

Check:

- `JWT_SECRET`
- `SECRET_ENCRYPTION_KEY`
- database password inputs

### Browser can load the UI but API requests fail

Common cause: `CORS_ALLOWED_ORIGINS` does not include the actual UI origin.

Verify the exact scheme, host, and port used by the browser.

### Quickstart values leaked into a real environment

Common cause: an operator reused the quickstart file as a base for staging or production.

Fix by rotating:

- JWT signing secret
- secret-encryption key
- database password

Then replace the quickstart file with environment-specific secrets management.
