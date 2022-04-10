/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	//"sigs.k8s.io/controller-runtime/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	simplegroupv0 "mihai.domain/simpledeployment/api/v0"
)

// SimpleDeploymentReconciler reconciles a SimpleDeployment object
type SimpleDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=simplegroup.mihai.domain,resources=simpledeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=simplegroup.mihai.domain,resources=simpledeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=simplegroup.mihai.domain,resources=simpledeployments/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SimpleDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SimpleDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("simpleDeployment", req.NamespacedName)

	log.V(1).Info("Enter Reconcile() with", "request", req)

	var simpleDeployment = &simplegroupv0.SimpleDeployment{}
	if err := r.Get(ctx, req.NamespacedName, simpleDeployment); err != nil {
		log.Error(err, "Unable to fetch SimpleDeployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(1).Info("Got simpleDeployment", "sdSpec", simpleDeployment.Spec)

	// --------------------------------------------------------------
	// Name the finalizer for TLS certs created by Cert-Manager
	tlsSecretFinalizer := "simplegroup.mihai.domain/tls-cert"
	// examine DeletionTimestamp to determine if object is under deletion
	if simpleDeployment.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(simpleDeployment, tlsSecretFinalizer) {
			controllerutil.AddFinalizer(simpleDeployment, tlsSecretFinalizer)
			if err := r.Update(ctx, simpleDeployment); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(simpleDeployment, tlsSecretFinalizer) {
			if err := r.deleteExternalResources(ctx, simpleDeployment); err != nil {
				// if fail to delete the external dependency here, return with error so that it can be retried
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(simpleDeployment, tlsSecretFinalizer)
			if err := r.Update(ctx, simpleDeployment); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// ---------------------------------------------------------------

	// Build the deployment that we would want to see exist within the cluster
	deployment := setupMinimalDeployment(simpleDeployment)
	// Set the controller reference, specifying that this Deployment is controlled by the SimpleDeployment being reconciled.
	// This will allow for the SimpleDeployment to be reconciled when changes to the Deployment are noticed.
	if err := controllerutil.SetControllerReference(simpleDeployment, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Manage your Service.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	if err := r.reconcileDeployment(ctx, simpleDeployment, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// Build the service that we would want to see exist within the cluster
	service := setupMinimalService(simpleDeployment)
	// Set the controller reference, specifying that this Service is controlled by the SimpleDeployment being reconciled.
	// This will allow for the SimpleDeployment to be reconciled when changes to the Service are noticed.
	if err := controllerutil.SetControllerReference(simpleDeployment, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Manage your Service.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	if err := r.reconcileService(ctx, simpleDeployment, service); err != nil {
		return ctrl.Result{}, err
	}

	// Build the ingress that we would want to see exist within the cluster
	ingress := setupMinimalIngress(simpleDeployment)
	// Set the controller reference, specifying that this Ingress is controlled by the SimpleDeployment being reconciled.
	// This will allow for the SimpleDeployment to be reconciled when changes to the Ingress are noticed.
	if err := controllerutil.SetControllerReference(simpleDeployment, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Manage your Ingress.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	if err := r.reconcileIngress(ctx, simpleDeployment, ingress); err != nil {
		return ctrl.Result{}, err
	}

	// Update SD status if necessary
	var upd []string
	dn := deployment.Namespace + "/" + deployment.Name
	for b := true; b; {
		switch {
		case simpleDeployment.Status.Deployment != dn:
			simpleDeployment.Status.Deployment = dn
			upd = append(upd, dn)
		case simpleDeployment.Status.URL != deriveURL(simpleDeployment):
			simpleDeployment.Status.URL = deriveURL(simpleDeployment)
			upd = append(upd, deriveURL(simpleDeployment))
		default:
			b = false
		}
	}
	if len(upd) > 0 {
		if err := r.Status().Update(ctx, simpleDeployment); err != nil {
			log.Error(err, "unable to update SimpleDeployment status")
			//return ctrl.Result{RequeueAfter: time.Second * 10}, err
			return ctrl.Result{}, err
		}
	}

	// Reconcile periodically
	return ctrl.Result{RequeueAfter: 15 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simplegroupv0.SimpleDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}

// Mihai - func to delete external resources due to finalizer for TLS secret
func (r *SimpleDeploymentReconciler) deleteExternalResources(ctx context.Context, sd *simplegroupv0.SimpleDeployment) error {
	nsdname := types.NamespacedName{Name: deriveName(sd, &netv1.IngressTLS{}), Namespace: sd.Namespace}
	log := r.Log.WithValues("finalizer", nsdname)
	log.V(1).Info("Enter deleteExternalResources()")

	foundSecret := &corev1.Secret{}
	err := r.Get(ctx, nsdname, foundSecret)

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("TLS Secret was not found so we can exit to remove the finalizer from SD.")
		return nil
	} else if err == nil {
		log.V(1).Info("Found TLS Secret", "foundSecretUID", foundSecret.GetUID())
		// pbly need to add logic to make sure Ingress is deleted/doesn't exist first
		err = r.Delete(ctx, foundSecret)
		if err != nil {
			log.Error(err, fmt.Sprintf("Unable to delete TLS Secret %s", nsdname))
			return err
		}
	}
	return nil
}

// Mihai - func to manage Service
func (r *SimpleDeploymentReconciler) reconcileDeployment(ctx context.Context, sd *simplegroupv0.SimpleDeployment, deployment *appsv1.Deployment) error {
	nsdname := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
	log := r.Log.WithValues("deployment", nsdname)
	log.V(1).Info("Enter reconcileDeployment()")
	// Manage your Deployment.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	// I'm not addressing the need to delete deplyoment and recreate in case of selector changes.
	foundDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, nsdname, foundDeployment)

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating new Deployment", "Deployment", deployment.Name, "Spec", deployment.Spec)
		err = r.Create(ctx, deployment)
		if err != nil {
			log.Error(err, "Unable to create new Deployment.")
			return err
		}
	} else if err == nil {
		log.V(1).Info("Found Deployment", "foundDeploSpec", foundDeployment.Spec, "foundDeploStatus", foundDeployment.Status)
		var upd []string // record need for update
		if *foundDeployment.Spec.Replicas != *deployment.Spec.Replicas {
			foundDeployment.Spec.Replicas = deployment.Spec.Replicas
			upd = append(upd, fmt.Sprintf("Replicas want %d got %d", deployment.Spec.Replicas, foundDeployment.Spec.Replicas))
		}
		for i, csp := range foundDeployment.Spec.Template.Spec.Containers {
			if csp.Name == deployment.Spec.Template.Name && csp.Image != sd.Spec.Image {
				foundDeployment.Spec.Template.Spec.Containers[i].Image = sd.Spec.Image
				upd = append(upd, fmt.Sprintf("Image want %s got %s", sd.Spec.Image, csp.Image))
			}
		}
		if len(upd) > 0 {
			log.V(1).Info("Need to Update existing Deployment", "updates", upd)
			err = r.Update(ctx, foundDeployment)
			if err != nil {
				log.Error(err, "Unable to update existing Deployment.")
				return err
			}
		}

	}
	return nil
}

// Mihai - func to manage Service
func (r *SimpleDeploymentReconciler) reconcileService(ctx context.Context, sd *simplegroupv0.SimpleDeployment, service *corev1.Service) error {
	nsdname := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	log := r.Log.WithValues("service", nsdname)
	log.V(1).Info("Enter reconcileService()")
	// Manage your Service.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	foundService := &corev1.Service{}
	err := r.Get(ctx, nsdname, foundService)

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating new Service", "Service", service.Name, "Spec", service.Spec)
		err = r.Create(ctx, service)
		if err != nil {
			log.Error(err, "Unable to create new Service.")
			return err
		}
	} else if err == nil {
		log.V(1).Info("Found Service", "foundSvcSpec", foundService.Spec, "foundSvcStatus", foundService.Status)
		var upd []string // record need for update of service
		for k, v := range setupSelectionLabels(sd) {
			if val, ok := foundService.Spec.Selector[k]; !ok || val != v {
				foundService.Spec.Selector[k] = v
				upd = append(upd, fmt.Sprintf("labelSelector %s=%s", k, v))
			}
		}
		if len(upd) > 0 {
			log.V(1).Info("Need to Update existing Service", "updates", upd)
			err = r.Update(ctx, foundService)
			if err != nil {
				log.Error(err, "Unable to update existing Service.")
				return err
			}
		}
	}

	return nil
}

// Mihai - func to manage Ingress resource
func (r *SimpleDeploymentReconciler) reconcileIngress(ctx context.Context, sd *simplegroupv0.SimpleDeployment, ingress *netv1.Ingress) error {
	nsdname := types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}
	log := r.Log.WithValues("ingress", nsdname)
	log.V(1).Info("Enter reconcileIngress()")
	// Manage your Service.
	// Create it if it doesn't exist. Update it if it is configured incorrectly.
	foundIngress := &netv1.Ingress{}
	err := r.Get(ctx, nsdname, foundIngress)

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating new Ingress", "Ingress", ingress.Name, "Spec", ingress.Spec)
		err = r.Create(ctx, ingress)
		if err != nil {
			log.Error(err, "Unable to create new Ingress.")
			return err
		}
	} else if err == nil {
		log.V(1).Info("Found Ingress", "foundIngrSpec", foundIngress.Spec, "foundIngrStatus", foundIngress.Status)
		var upd []string // record need for update of service
		for k, v := range ingress.Labels {
			if val, ok := foundIngress.Labels[k]; !ok || val != v {
				foundIngress.Labels[k] = v
				upd = append(upd, fmt.Sprintf("label %s=%s", k, v))
			}
		}
		// deal with the rewrite annotation
		annot := foundIngress.ObjectMeta.GetAnnotations()
		if a, ok := annot["nginx.ingress.kubernetes.io/rewrite-target"]; !ok || a != "/" {
			foundIngress.ObjectMeta.SetAnnotations(map[string]string{"nginx.ingress.kubernetes.io/rewrite-target": "/"})
			upd = append(upd, "Annotation rewrite-target=true")
		}
		if len(upd) > 0 {
			log.V(1).Info("Need to Update existing Ingress", "updates", upd)
			err = r.Update(ctx, foundIngress)
			if err != nil {
				log.Error(err, "Unable to update existing Ingress.")
				return err
			}
		}
	}
	return nil
}

// Mihai - helper for almost-minimal Deployment structure
func setupMinimalDeployment(sd *simplegroupv0.SimpleDeployment) *appsv1.Deployment {
	name := deriveName(sd, &appsv1.Deployment{})
	labels := setupSelectionLabels(sd)
	// Build the deployment that we want to see exist within the cluster
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sd.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: sd.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: sd.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Mihai - helper for almost-minimal Service structure
func setupMinimalService(sd *simplegroupv0.SimpleDeployment) *corev1.Service {
	name := deriveName(sd, &corev1.Service{}) // didn't treat the error case of ""
	labels := setupSelectionLabels(sd)
	// Build the Service that we want to see exist within the cluster
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sd.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: setupSelectionLabels(sd),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

// Mihai - helper for almost-minimal Ingress structure
func setupMinimalIngress(sd *simplegroupv0.SimpleDeployment) *netv1.Ingress {
	sdIngr := sd.Spec.IngressInfo
	// Build a few default items
	name := deriveName(sd, &netv1.Ingress{})
	labels := setupSelectionLabels(sd)
	exactPathType := netv1.PathTypeExact
	// Build default annotations
	annot := make(map[string]string)
	deriveIngressAnnotations(sd, annot)

	// Build the deployment that we want to see exist within the cluster
	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   sd.Namespace,
			Labels:      labels,
			Annotations: annot,
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &sd.Spec.IngressInfo.IngressClassName,
			TLS: []netv1.IngressTLS{
				{
					Hosts:      []string{sdIngr.Host},
					SecretName: deriveName(sd, &netv1.IngressTLS{}),
				},
			},
			Rules: []netv1.IngressRule{
				{
					Host: sdIngr.Host,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     sdIngr.Path,
									PathType: &exactPathType,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: deriveName(sd, &corev1.Service{}),
											Port: netv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return ingress
}

// Mihai - helper for backend service name
func deriveName(sd *simplegroupv0.SimpleDeployment, t interface{}) string {
	var suff string
	switch t.(type) {
	case *appsv1.Deployment:
		suff = "-deplo"
	case *corev1.Service:
		suff = "-svc"
	case *netv1.Ingress:
		suff = "-ingr"
	case *netv1.IngressTLS:
		suff = "-ingr-tls"
	default:
		return ""
	}
	return sd.Name + suff
}

// Mihai - helper for LabelSelector
func setupSelectionLabels(sd *simplegroupv0.SimpleDeployment) map[string]string {
	return map[string]string{
		"sd-member": "true",
		"sd":        sd.Name,
		"app":       "nginx-oper",
	}
}

// Mihai - helper for Ingress Annotations
// pbly better as a method in of simplegroupv0.SimpleDeployment
func deriveIngressAnnotations(sd *simplegroupv0.SimpleDeployment, annot map[string]string) {
	// Add annotation for cert-manager cluster-issuer with default name for this project
	annot["cert-manager.io/cluster-issuer"] = "ca-issuer"

	// Add annotation for RewriteTarget based on IngressController configured in Spec
	// For empty string assume a rewrite is not necessary
	if sd.Spec.IngressInfo.RWTarget == "" {
		return
	}
	switch sd.Spec.IngressInfo.IngressControllerType {
	case "nginx":
		annot["nginx.ingress.kubernetes.io/rewrite-target"] = sd.Spec.IngressInfo.RWTarget
	default:
		annot["unsupported-ingressControllerType"] = "true"
	}
	return
}

// Mihai - helper to compose URL reported in Status
// this should probably be a Method of the simplegroupv0.SimpleDeployment type
func deriveURL(sd *simplegroupv0.SimpleDeployment) string {
	sp := sd.Spec.IngressInfo
	var url string
	switch sp.PublicPort {
	case 443:
		url = fmt.Sprintf("https://%s%s", sp.Host, sp.Path)
	default:
		url = fmt.Sprintf("https://%s:%d%s", sp.Host, sp.PublicPort, sp.Path)
	}
	return url
}
