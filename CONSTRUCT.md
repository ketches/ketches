# 项目构建过程

## 初始化项目

```bash
mkdir ketches
cd ketches

kubebuilder init --domain ketches.io --repo github.com/ketches/ketches --owner "The Ketches Authors"
kubebuilder edit --multigroup
```

## 创建 API

```bash
kubebuilder create api --group core --version v1alpha1 --kind Application --resource --controller
```