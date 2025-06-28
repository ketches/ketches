# Ketches 后端 API 环境变量配置说明

本项目后端服务（ketches-api）支持通过环境变量进行全部配置，推荐在 Kubernetes、Docker Compose 或本地开发时使用。

## 必需/常用环境变量

| 变量名         | 说明                              | 默认值（如有）                                      |
|:--------------|:-----------------------------------|:---------------------------------------------------|
| APP_HOST      | 服务监听地址                      | 0.0.0.0                                            |
| APP_PORT      | 服务监听端口                      | 8080                                               |
| APP_RUNMODE   | 运行模式（dev/prod等）            | dev                                                |
| APP_JWT_SECRET| JWT签名密钥                       | ketches                                            |
| DB_TYPE       | 数据库类型（postgres/mysql/sqlite）| sqlite                                             |
| DB_DNS        | 数据库连接字符串                  | file:ketches.db?cache=shared&mode=rwc（sqlite默认） |

## PostgreSQL 示例

```env
DB_TYPE=postgres
DB_DNS="host=postgres port=5432 user=postgres password=postgres dbname=ketches sslmode=disable"
```

## SQLite 示例（默认）

```env
DB_TYPE=sqlite
DB_DNS="file:ketches.db?cache=shared&mode=rwc"
```

## MySQL 示例

```env
DB_TYPE=mysql
DB_DNS="ketches:ketches@tcp(mysql:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local"
```

## 其它说明

- 所有环境变量均可通过 Docker/K8s 的 environment 字段注入。
- 若未设置，程序将使用默认值。
- 生产环境请务必修改 APP_JWT_SECRET。

如需更多自定义配置，请参考源码 `backend/internal/app/config.go`。
