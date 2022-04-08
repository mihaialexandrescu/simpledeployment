#!/bin/bash

set -euo pipefail

# Requirements:
# - kubectl
# - helm3 ? not yet
# - kind

EX_DIR=examples-deploy/kind-with-lb

# Start a Kind cluster
KIND_NAME=kind
KIND_CFG=${EX_DIR}/kind-config-01.yaml
kind create cluster --name ${KIND_NAME} --config ${KIND_CFG}


# Add MetalLB
METALLB_VERSION=v0.12.1
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/${METALLB_VERSION}/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/${METALLB_VERSION}/manifests/metallb.yaml
kubectl apply -f ${EX_DIR}/metallb-configmap.yaml


# Add an nginx ingress controller
NGINX_INGRESS_CONTROLLER_VERSION=v1.1.2
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-${NGINX_INGRESS_CONTROLLER_VERSION}/deploy/static/provider/cloud/deploy.yaml

kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s


# Create CA key that will be used for the CA ClusterIssuer resource of CertManager
TMP_CERT_DIR=${EX_DIR}/tmp
openssl req -newkey rsa:2048 -nodes -keyout ${TMP_CERT_DIR}/tls.key -subj "/CN=CERTMNGR-K8S-CA/C=RO/O=KIND/OU=CLUSTER" -x509 -days 30 -out ${TMP_CERT_DIR}/tls.crt


# Deploy CertManager
CERTMANAGER_VERSION=v1.8.0
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERTMANAGER_VERSION}/cert-manager.yaml

kubectl wait --namespace cert-manager \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/instance=cert-manager \
  --timeout=90s


# Upload CA Key and create ClusterIssuer Resource (clusterissuer is a cluster scoped resource)
kubectl apply --namespace cert-manager -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ca-key-pair
data:
  tls.crt: $(cat ${TMP_CERT_DIR}/tls.crt | base64 -w0)
  tls.key: $(cat ${TMP_CERT_DIR}/tls.key | base64 -w0)
EOF

kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: ca-key-pair
EOF


sleep 10s

# Local build example and then deploy operator using kustomize
#make docker-build IMG=simpledeployment:v0.1.0
#kind load docker-image simpledeployment:v0.1.0 --name kind
#kubectl apply -k config/default


# Local build example and then deploy operator using helm
make docker-build IMG=simpledeployment:v0.1.0
kind load docker-image simpledeployment:v0.1.0 --name kind
helm upgrade --install --timeout=30s --wait \
	--set image.repository=simpledeployment --set image.tag=v0.1.0 \
	--namespace simpledeployment-system --create-namespace \
	simpledeployment-release charts/simpledeployment-operator


sleep 5s

# Deploy examples
#kubectl apply -f examples-cr/

#sleep 30s

#kubectl get deployment,svc,secret,ingress,sd -o wide



# Optional instructions for DNS
LB_IP=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "${LB_IP} is the LB address allocated to the IngressController."

#echo "curl --cacert ${TMP_CERT_DIR}/tls.crt https://<...>"

#for VAR in $(kubectl get sd -A -o jsonpath='{.items[*].status.url}')
#do
#  curl --cacert $MYCACERT $VAR
#done
