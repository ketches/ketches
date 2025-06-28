# Ketches Backend Environment Variables

The backend service (ketches-api) can be fully configured via environment variables. This is recommended for Kubernetes, Docker Compose, or local development.

## Required/Common Environment Variables

| Variable        | Description                        | Default (if any)                                   |
|:---------------|:-----------------------------------|:---------------------------------------------------|
| APP_HOST        | Service listen address             | 0.0.0.0                                            |
| APP_PORT        | Service listen port                | 8080                                               |
| APP_RUNMODE     | Run mode (dev/prod, etc.)          | dev                                                |
| APP_JWT_SECRET  | JWT signing secret                 | ketches                                            |
| DB_TYPE         | Database type (postgres/mysql/sqlite) | sqlite                                         |
| DB_DNS          | Database connection string         | file:ketches.db?cache=shared&mode=rwc (sqlite)     |

## PostgreSQL Example

```env
DB_TYPE=postgres
DB_DNS="host=postgres port=5432 user=postgres password=postgres dbname=ketches sslmode=disable"
```

## SQLite Example (default)

```env
DB_TYPE=sqlite
DB_DNS="file:ketches.db?cache=shared&mode=rwc"
```

## MySQL Example

```env
DB_TYPE=mysql
DB_DNS="ketches:ketches@tcp(mysql:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local"
```

## Notes

- All variables can be injected via Docker/K8s `environment` fields.
- If not set, defaults will be used.
- For production, be sure to change `APP_JWT_SECRET`.

For more details, see `backend/internal/app/config.go`.
