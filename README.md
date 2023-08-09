# 该代码库已停止更新，并将在未来的某个时间被移除，其代码已被合并至 [Nautes](https://github.com/nautes-labs/nautes)。

# Argo Operator
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![golang](https://img.shields.io/badge/golang-v1.20.0-brightgreen)](https://go.dev/doc/install)
[![version](https://img.shields.io/badge/version-v0.3.6-green)]()

Argo Operator 项目提供了一组用于调谐 Cluster 资源和 CodeRepo 资源事件的 Controller，调谐内容主要是将 Cluster 资源所声明的 Kubernetes 集群和 CodeRepo 资源所声明的代码库同步到同集群的 ArgoCD 中，使 ArgoCD 中使用了这些 Kubernetes 集群和代码库的 Application 可以正常工作。

## 功能简介

Controller 除了可以通过响应资源的增删改事件进行调谐外，还以定时轮询的方式检查 Kubernetes 集群和代码库信息的变化并进行调谐。

当 Controller 监听到 Cluster 资源有增改或检测到密钥管理系统中的 Kubernetes 集群 kubeconfig 信息有变化时，会主动从密钥管理系统中获取最新的 kubeconfig 内容，并通过 API 将集群同步到 ArgoCD 中。同步成功后 Controller 会在 Cluster 资源中记录此次同步的 kubeconfig 内容的版本，作为下一次同步时检查数据差异的基准值。当 Controller 监听到 Cluster 资源被删除时，会通过 API 从 ArgoCD 中移除相应的 Kubernetes 集群。

当 Controller 监听到 CodeRepo 资源有增改或检测到密钥管理系统中的代码库 DeployKey 信息有变化时，会主动从密钥管理系统中获取最新的 DeployKey 内容，并通过 API 将代码库同步到 ArgoCD 中。同步成功后 Controller 会在 CodeRepo 资源中记录此次同步的 DeployKey 内容的版本，作为下一次同步时检查数据差异的基准值。当 Controller 监听到 CodeRepo 资源被删除时，会通过 API 从 ArgoCD 中移除相应的代码库。

## 快速开始

### 准备

安装以下工具，并配置 GOBIN 环境变量：

- [go](https://golang.org/dl/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

准备一个 kubernetes 实例，复制 kubeconfig 文件到 {$HOME}/.kube/config

创建 nautes-configs 配置文件

```
kubectl create cm nautes-configs -n nautes
```

### 构建

```
make build
```

### 运行

```bash
make run
```

### 单元测试

执行单元测试

```shell
make test
```

