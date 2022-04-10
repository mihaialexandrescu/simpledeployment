# simpleDeployment Operator

## Project Summary

This `simpleDeployment` Operator is meant to deploy an application (currently targeted at a vanilla `nginx`) on top of Kubernetes using a CustomResource.

The application is meant to be accessed via HTTPS.

This project is based on [kubebuilder](https://book.kubebuilder.io/) v3.

Development has been done on a local Kind cluster with an nginx Ingress Controller and a MetalLb load balancer.


## Prerequisites, Assumptions and Limitations

* A running Kubernetes cluster and `kubectl`

* The [`nginx` Ingress Controller](https://kubernetes.github.io/ingress-nginx/) is running in the cluster.
  * To deploy it you can use the quickstart guide [here](https://kubernetes.github.io/ingress-nginx/deploy/)
  * To expose the Ingress Controller outside the Kubernetes cluster you can use a Service of type LoadBalancer (for instance locally with MetalLB) or a NodePort.
  * If the public port of the Ingress Controller has been changed from 443, it must be also configured in the CustomResource Spec to be accoundted for by the Operator.

* To manage TLS certificates, the Operator assumes:
  * there is a `Cert-Manager` instance running
  * that a `ClusterIssuer` cluster-scoped resource named `ca-issuer` has been configured
  * that a CA Certificate and CA Key have been uploaded in a Secret inside the same namespace as the Cert-Manager Pods and is referenced by the `ClusterIssuer` according to [this docs page](https://cert-manager.io/docs/configuration/ca/).
    * a working example for local development is present in the [`examples-deploy/kind-with-lb/start.sh`](examples-deploy/kind-with-lb/start.sh) file (along other example setup steps)

* SSL Passthrough (to extend HTTPS to the Pod) is currently not supported (missing annotation and certificate management logic for the application's Pods).

* There isn't a Helm repo; just the local `charts/` directory in this repo.
  * If Helm is the chosen means of installation then a `helm` installation (only testd on Helm3) is also necessary (https://helm.sh/docs/intro/install/)



## Configuration/User input for the CustomResource

The `SimpleDeployment` CustomResource is Namespaced and its structure is illustrated below:
```
apiVersion: simplegroup.mihai.domain/v0
kind: SimpleDeployment
metadata:
  name: <...>
  namespace: <...>
spec:
  ... ...
  < spec components are explained below >
```

The `spec` of these CustomResources has the following elements:

* `image`
  * Container image to use including the tag
  * Mandatory and does not default
  * Example: `nginx:latest`
* `replicas`
  * Number of Replicas for the Deployment object for the app
  * Optional and defaults to 1
  * Users can set it to 0 to effectively disable their app but this will not delete the helper resources (like Service or Ingress)
* `ingressInfo`
  * Info on where the application will be published; the elements will be used for the Ingress rules that will be configured by the Operator
* `ingressInfo.IngressControllerType`
  * Used to determine Annotation values for varios Ingress Controller features
  * Only predetermined values can be used (it is an ENUM) and the only currently supported value is `nginx`.
  * Defaults to `nginx`
* `ingressInfo.IngressClassName`
  * Used to select which IngressController will be used by the Ingress the operator defines
  * Can be optional if there is a default Ingress Controller defined in the K8s cluster
  * Defaults to `nginx`
* `ingressInfo.publicPort`
  * Port opened for this app on the Ingress Controller (prerequisite)
  * This is NOT the port specified in the ingress rule for the backend service
  * Must be configured in correspondence with the configuration of the Service that exposes the Ingress Controller
  * Defaults to `443`
* `host`
  * Maps to the Host HTTP Header that will need to be used to access the app
  * Is used to build the Ingress rule
  * Optional if host matching is not required
  * Example: `dev.local`
* `path`
  * Is used to build the Ingress rule
  * Optional and defaults to `/`
  * Should always start with `/`
* `rewriteTarget`
  * Holds the expression to be used to configure the RewriteTarget feature on the Ingress Controller
  * Optional and defaults to an empty string indicating that the feature is not needed

Example:
```
apiVersion: simplegroup.mihai.domain/v0
kind: SimpleDeployment
metadata:
  name: sd1
spec:
  replicas: 1
  image: nginx:latest
  ingressInfo:
    ingressControllerType: nginx
    ingressClassName: nginx
    host: dev.local
    publicPort: 443
    path: /sdp1
    rewriteTarget: /
```

The following out illustrates the effect of the config above on some of the created objects (not all are displayed):
```
$ kubectl get sd
NAME   CFGIMAGE       CFGREPLICAS   DEPLOYMENT          URL                     AGE
sd1    nginx:latest   1             default/sd1-deplo   https://dev.local/sdp1   55s


$ kubectl get deployment/sd2-deplo -o wide
NAME        READY   UP-TO-DATE   AVAILABLE   AGE     CONTAINERS   IMAGES         SELECTOR
sd1-deplo   1/1     1            1           5m10s   web          nginx:latest   app=nginx-oper,sd=sd1,sd-member=true


$ kubectl get ingress/sd2-ingr -o yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    cert-manager.io/cluster-issuer: ca-issuer
    nginx.ingress.kubernetes.io/rewrite-target: /
  labels:
    app: nginx-oper
    sd: sd1
    sd-member: "true"
  name: sd1-ingr
  < ... >
spec:
  ingressClassName: nginx
  rules:
  - host: dev.local       <<<
    http:
      paths:
      - backend:
          service:
            name: sd1-svc
            port:
              name: http
        path: /sdp1         <<<
        pathType: Exact
  tls:
  - hosts:
    - dev.local            <<<
    secretName: < ... >
```



Examples of these configurations are provided in the `examples-cr/` [directory](examples-cr/).


## Install

You can deploy the `simpleDeployment` Operator with Kubernetes manifests (and kustomize) or with the provided [Helm chart](charts/simpledeployment-operator/) (only tested with Helm3) under the `charts` folder.

Either method will install the CRD, the RBAC rules and the Operator Deployment.

* Manifests with Kustomize:
```
# To inspect the manifests that will be installed
kubectl kustomize config/default

# To install them (the following command will install in the "simpledeployment-system" namespace)
kubectl apply --kustomize config/default
```

* Helm chart (with Helm3)
```
# In the "default" Namespace
helm upgrade --install --timeout=30s \
	simpledeployment-release charts/simpledeployment-operator

# In a specified Namespace (e.g. simpledeployment-system)
helm upgrade --install --timeout=30s \
	--namespace simpledeployment-system --create-namespace \
	simpledeployment-release charts/simpledeployment-operator
```


## Local Test Deployment with Kind

There is an example bash script in the `examples-deploy/` [folder](examples-deploy/) for a local deployment including prerequisites. This uses Kind, MetalLB (for the LoadBalancer Service), Cert-Manager in its default namespace ("cert-manager") and a locally generated CA key pair and self signed certificate uploaded to Kubernetes for the ClusterIssuer.

The `curl`s performed for checking at the end (if uncommented) assume a DNS entry for the configured `host` ingress rules but that can be avoided by using the `-H 'Host: <host>'` option if you want to target the LB IP.
