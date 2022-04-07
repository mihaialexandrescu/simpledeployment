# simpleDeployment Operator task

This repo is meant to hold an operator that deploys an application (currently targeted at a vanilla nginx) on top of Kubernetes using a custom resource and an operator for it.

The application is meant to be accessed via HTTPS.

Development done on a local Kind cluster with an nginx Ingress Controller and a MetalLb load balancer.  
This project is based on kubebuilder (as is obvious from the file structure).
