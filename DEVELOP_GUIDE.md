# Develop Guide

## Controller Manager 项目

### 生成 client-go

包含 clientset、informer、listers 等代码。

当修改了 api 之后，重新生成 client-go 代码，执行以下命令：

```bash
make update-codegen
```

当修改了 api 之后，重新生成并验证 client-go 代码，执行以下命令：

```bash
make update-codegen-verify
```

## Http Server 项目

### 生成 swagger 文档

安装：

- swag：`go install github.com/swaggo/swag/cmd/swag@latest`

生成：

```bash
make swag-init
```

### 运行

在运行之前，需要确保已经对目标集群运行 Controller Manager。

```bash
go run cmd/api-server/main.go -jwt-secret=ketches
```