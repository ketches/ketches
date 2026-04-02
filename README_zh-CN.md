<div align="center">

<img src="ui/public/ketches.svg" alt="Ketches Logo" width="80" height="80" />

# Ketches

**让 Kubernetes 应用管理变得简单的云原生平台。**

[![CI](https://github.com/ketches/ketches/actions/workflows/ci.yml/badge.svg)](https://github.com/ketches/ketches/actions/workflows/ci.yml)
[![Release](https://github.com/ketches/ketches/actions/workflows/release.yml/badge.svg)](https://github.com/ketches/ketches/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/ketches/ketches)](go.mod)
[![License](https://img.shields.io/github/license/ketches/ketches)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/ketches/ketches)](https://github.com/ketches/ketches/releases)

[English](README.md) · [简体中文](README_zh-CN.md)

</div>

---

## 项目简介

Ketches 是一个面向企业级的开源云原生应用平台，旨在降低 Kubernetes 的使用门槛。通过直观的 Web 界面和完善的 REST API，帮助开发团队快速将应用构建、部署并管理到多个 Kubernetes 集群，无需深入了解 Kubernetes 底层概念。

## ✨ 核心功能

| 功能模块 | 亮点 |
| ------- | ---- |
| **多集群管理** | 通过 KubeConfig 接入并统一管理多个 Kubernetes 集群 |
| **项目与环境** | 项目 → 环境 → 应用三级隔离模型，每个环境对应独立的 K8s Namespace |
| **应用生命周期** | 支持部署、启动、停止、重新部署、版本回滚及调试模式 |
| **实时运维** | 按 Pod 提供实时日志流和浏览器内 WebSocket 终端连接 |
| **网关管理** | 基于 Gateway API 配置 HTTP / HTTPS / TCP / UDP 入口规则 |
| **资源管理** | CPU / 内存限制、持久卷（PVC）、基于 ConfigMap 的配置文件挂载 |
| **健康检查** | Liveness、Readiness、Startup 三类探针，支持 HTTP / TCP / Exec 方式 |
| **调度规则** | 节点选择器、节点亲和性、污点容忍配置 |
| **权限控制** | 系统级（admin / user）与项目级（owner / developer / viewer）双层 RBAC |
| **运行时数据库** | 默认 PostgreSQL，可选 MySQL，测试使用纯 Go SQLite |
| **集群扩展** | 通过 Helm 安装和管理集群扩展（Gateway API、监控组件等） |

## 🏗️ 系统架构

```txt
┌──────────────────────────────────────────────────────────────────┐
│                           用户浏览器                              │
└──────────────────────────────────────────────────────────────────┘
                                │ HTTPS
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Nginx / Ingress                             │
│                    （反向代理 & 静态资源）                          │
└───────────────────────┬──────────────────────┬───────────────────┘
                        │ /                    │ /api/*
                        ▼                      ▼
        ┌───────────────────────┐   ┌──────────────────────────┐
        │  ketches-ui（前端）    │   │  ketches API（Go/Gin）    │
        │  shadcn/ui · Zustand  │   │  路由 → Handler           │
        │  TanStack Query       │   │  → Service → Core        │
        └───────────────────────┘   └────────────┬─────────────┘
                                                 │
                        ┌────────────────────────┼──────────────────────┐
                        ▼                        ▼                      ▼
             ┌──────────────────┐    ┌────────────────────┐  ┌────────────────────┐
             │     数据库        │    │   Kubernetes       │  │   Kubernetes       │
             │ PostgreSQL /     │    │   集群 1            │  │   集群 N           │
             │ MySQL            │    │   (client-go)      │  │   (client-go)      │
             └──────────────────┘    └────────────────────┘  └────────────────────┘
```

**后端分层**：路由层 → 中间件层（Auth / CORS / RBAC）→ Handler 层 → Service 层 → Core 层（K8s 资源构建）→ Kube Client 层（client-go）→ DB 层（GORM）

## 🚀 快速开始

### Docker Compose（推荐）

**前置条件**：Docker 24+ 并已启用 Compose V2

```bash
curl -fsSL https://raw.githubusercontent.com/ketches/ketches/master/deploy/docker/docker-compose.yml -o docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/ketches/ketches/master/deploy/docker/.env.quickstart -o .env
docker compose up -d
```

这条 QuickStart 路径仅用于本地试用。它会使用 `deploy/docker/.env.quickstart` 中的演示密钥，方便快速启动。正式环境请保持 `deploy/docker/docker-compose.yml` 的安全默认行为，并参考[生产部署说明](docs/PRODUCTION_DEPLOYMENT.md)。

默认管理员账号在首次启动时自动创建，查看日志获取：

```bash
docker compose logs ketches-api | grep -i "admin"
```

### Helm（Kubernetes）

**前置条件**：Kubernetes 1.24+、Helm 3.12+

```bash
helm upgrade --install ketches ./deploy/helm/ketches --namespace ketches --create-namespace -f ./deploy/helm/ketches/values-quickstart.yaml
```

这条 QuickStart 路径仅适用于本地体验集群。主 Helm Chart 仍然保持安全默认值，需要在非体验场景下显式提供真实密钥。

### 原始 Kubernetes manifests

**前置条件**：Kubernetes 1.24+

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/ketches/master/deploy/kubernetes/manifests.quickstart.yaml
```

这份 quickstart manifest 仅用于本地体验。它内置了演示密钥和面向 localhost 的 CORS 配置，使原始 manifest 的体验路径与其他 quickstart 方式一致。

更多说明见：[`deploy/kubernetes/README.md`](deploy/kubernetes/README.md)

QuickStart 下 UI Service 类型为 `NodePort`（`30080`）：

```bash
kubectl -n ketches get svc ketches-ui
```

也可以使用端口转发访问：

```bash
kubectl -n ketches port-forward svc/ketches-ui 8080:80
# 浏览器打开 http://127.0.0.1:8080
```

生产环境请不要复用 quickstart values 文件。请务必为 `config.jwtSecret` 和 `config.secretEncryptionKey` 提供强随机值；如果继续使用内置 PostgreSQL，还需要设置 `postgres.auth.password`，并参考[生产部署说明](docs/PRODUCTION_DEPLOYMENT.md)。如果使用外部数据库，请设置 `postgres.enabled=false`，并显式提供 `config.dbSource`（或拆分后的 `config.db*` 参数）。

#### Helm Chart 仓库（GitHub Pages）

仓库已将 Helm Chart 自动发布集成到 `.github/workflows/release.yml`。

当推送符合 `v*` 的 tag 时，GitHub Actions 会自动打包 `deploy/helm/ketches`，更新 `index.yaml`，并发布到 `gh-pages` 分支。

一次性配置步骤：

1. 进入仓库 GitHub Settings 的 **Pages**。
2. **Source** 选择 **Deploy from a branch**。
3. 分支选择 **gh-pages**，目录选择 **/**。
4. 保存。

之后即可通过 GitHub Pages 使用：

```bash
helm repo add ketches https://ketches.github.io/ketches
helm repo update
helm search repo ketches
helm install ketches ketches/ketches -n ketches --create-namespace
```

### 从源码构建

**前置条件**：Go 1.25+、Node.js 22+

```bash
git clone https://github.com/ketches/ketches.git
cd ketches

# 复制并编辑环境配置
cp .env.example .env

# 编译后端
make build

# 编译前端
make build-ui

# 启动后端（监听 :8080）
make run
# 另开终端启动前端开发服务器
make dev-ui
```

## ⚙️ 配置说明

所有配置项均通过环境变量设置；`.env` 文件仅作为本地开发时的可选方式。

| 变量名 | 默认值 | 说明 |
| ------ | ------ | ---- |
| `PORT` | `8080` | API 服务监听端口 |
| `LOG_LEVEL` | `info` | 日志级别（`debug` / `info` / `warn` / `error`） |
| `DB_DRIVER` | `postgres` | 运行时数据库驱动（`postgres` / `mysql`） |
| `DB_SOURCE` | *（可选）* | 完整数据库连接字符串；设置后优先于拆分数据库变量 |
| `DB_HOST` | 按驱动决定 | 未设置 `DB_SOURCE` 时的数据库主机 |
| `DB_PORT` | 按驱动决定 | 未设置 `DB_SOURCE` 时的数据库端口 |
| `DB_NAME` | `ketches` | 未设置 `DB_SOURCE` 时的数据库名 |
| `DB_USERNAME` | 按驱动决定 | 未设置 `DB_SOURCE` 时的数据库用户名 |
| `DB_PASSWORD` | 空 | 未设置 `DB_SOURCE` 时的数据库密码 |
| `DB_SSLMODE` | `disable` | 未设置 `DB_SOURCE` 时 PostgreSQL 的 SSL 模式 |
| `DB_AUTO_MIGRATE` | `true` | 是否在数据库初始化时执行 GORM AutoMigrate |
| `JWT_SECRET` | *（必填）* | JWT Token 签名密钥 |
| `SECRET_ENCRYPTION_KEY` | *（必填）* | 用于静态加密敏感数据的密钥 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,...` | 允许跨域的源地址，逗号分隔 |

**Note**：生产环境建议设置 `DB_AUTO_MIGRATE=false`，并通过可控的迁移流程管理数据库结构变更。

SQLite 不再作为运行时支持的数据库，仅在测试中通过纯 Go 驱动保留，以便后端构建彻底去掉 CGO。

**PostgreSQL 示例**：

```bash
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ketches
DB_USERNAME=postgres
DB_PASSWORD=<db-password>
DB_SSLMODE=disable
```

**MySQL 示例**：

```bash
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=ketches
DB_USERNAME=root
DB_PASSWORD=<db-password>
```

如果你更希望自行维护完整连接串，也仍然可以直接设置 `DB_SOURCE`。

完整配置参见 [`.env.example`](.env.example)。

## 🛠️ 开发指南

### 前置工具

| 工具 | 版本要求 | 用途 |
| ---- | ------- | ---- |
| Go | 1.25+ | 后端开发 |
| Node.js | 22+ | 前端开发 |
| Docker | 24+ | 可选，用于容器化开发 |

### 常用 `make` 命令

```bash
make help          # 查看所有可用命令

# 编译
make build         # 编译 Go 二进制 → bin/ketches
make build-ui      # 编译前端 → ui/dist/
make build-all     # 同时编译前后端

# 开发
make run           # 启动 API 服务（读取 .env）
make dev-ui        # 启动 Vite 开发服务器（:5173）
make openapi       # 生成 openapi/openapi.json 和 openapi/openapi.yaml

# 测试
make test          # 启用竞态检测运行测试
make test-coverage # 在浏览器中查看覆盖率报告

# 代码检查
make lint          # golangci-lint（后端）
make lint-ui       # eslint（前端）

# Docker
make docker-build  # 本地构建 API + UI 镜像
make docker-buildx # 构建并推送多架构镜像（amd64 + arm64）
make up            # docker compose up -d --build
make down          # docker compose down
make logs          # docker compose logs -f
```

### 目录结构

```txt
ketches/
├── cmd/api/            # 程序入口
├── internal/
│   ├── api/            # 通用请求/响应工具
│   ├── app/            # 应用配置、JWT、常量
│   ├── core/           # Kubernetes 资源构建（Deployment、Service 等）
│   ├── db/             # GORM 实体定义与迁移
│   ├── handlers/       # HTTP Handler（按领域拆分）
│   ├── kube/           # Kubernetes 客户端存储
│   ├── middlewares/    # 认证、CORS 中间件
│   ├── models/         # 请求/响应模型
│   ├── routes/         # API 路由定义
│   └── services/       # 业务逻辑层
├── ui/                 # React + TypeScript 前端
│   ├── src/
│   │   ├── api/        # Axios API 客户端模块
│   │   ├── components/ # 功能组件
│   │   ├── pages/      # 页面级组件
│   │   ├── stores/     # Zustand 状态管理
│   │   └── hooks/      # 自定义 React Hooks
│   ├── Dockerfile
│   └── nginx.conf.template
├── docs/               # PRD 和技术设计文档
├── Dockerfile          # 后端镜像
├── docker-compose.yml
└── Makefile
```

## 🤝 参与贡献

欢迎任何形式的贡献！请按以下步骤进行：

1. **Fork** 仓库并从 `main` 分支创建新分支
2. **提交** 修改，使用清晰、描述性的 commit 信息
3. **测试** 修改内容 — 后端执行 `make test`，前端执行 `make lint-ui`
4. 发起 **Pull Request** — CI 通过后方可合并

### 代码规范

- **后端**：遵循 Go 标准风格；使用 `any` 而非 `interface{}`；代码注释使用英文
- **前端**：使用 shadcn/ui 组件；样式统一用 Tailwind CSS，禁止内联 CSS；代码注释使用英文

## 📦 Docker 镜像

每次发布 Release 时，会自动构建并推送多架构镜像（`linux/amd64` + `linux/arm64`）到 GitHub Container Registry：

```bash
# API 镜像
docker pull ghcr.io/ketches/ketches-api:latest

# UI 镜像
docker pull ghcr.io/ketches/ketches-ui:latest
```

浏览所有标签：[ghcr.io/ketches](https://github.com/orgs/ketches/packages)

## 📄 开源许可

Ketches 基于 [Apache 2.0 许可证](LICENSE) 开源发布。
