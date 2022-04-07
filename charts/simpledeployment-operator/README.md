# Homework simpledeployment Operator Helm Chart

This directory contains a Kubernetes Helm chart to deploy the operator built in this homework.

## Prerequisites

* Kubernetes 1.6+
* Helm 3

## Installing the Chart

### Fresh install

To install the chart on a fresh cluster, use the following:

```bash
helm repo add ??banzaicloud-stable ???https://kubernetes-charts.banzaicloud.com
helm upgrade --install --namespace sd-sys --create-namespace sd-oper .
```


## Configuration

The exposed configuration is very limited.

| Parameter                    | Description                                              | Default                      |
| ---------------------------- | -------------------------------------------------------- | ---------------------------- |
| `image.repository`           | Container image to use                                   | `simpledeployment????`       |
| `image.tag`                  | Container image tag to deploy operator in                | `.Chart.AppVersion`          |
| `replicaCount`               | k8s replicas                                             | `1`                          |
