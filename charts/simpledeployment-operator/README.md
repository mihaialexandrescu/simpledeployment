# Homework simpledeployment Operator Helm Chart

This directory contains a Kubernetes Helm chart to deploy the "simpleDeployment" Operator from this repo.

## Prerequisites

* Kubernetes 1.6+
* Helm 3

## Installing the Chart

### Fresh install

To install the chart on a fresh cluster, use the following (assuming you are in this folder):

```bash
helm upgrade --install --namespace simpledeployment-system --create-namespace sd-oper-release .
```

## Configuration

The current exposed configuration for the Operator components through `values.yaml` is limited.

| Parameter                    | Description                                                | Default                      |
| ---------------------------- | ---------------------------------------------------------- | ---------------------------- |
| `image.repository`           | Container image to use to run the Operator                 |                              |
| `image.tag`                  | Container image tag for the Operator                       | `.Chart.AppVersion`          |
| `replicaCount`               | For the Deployment managing the Pods running the Operator  | `1`                          |
