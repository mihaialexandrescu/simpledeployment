package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	netv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	simplegroupv0 "mihai.domain/simpledeployment/api/v0"
)

var _ = Describe("SimpleDeployment controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		sdName      = "test-sd1"
		sdNamespace = "default"
		sdImage     = "nginx:latest"
		sdReplicas  = int32(2)

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 500
	)

	var sd *simplegroupv0.SimpleDeployment
	var ctx context.Context
	// defining the next 2 vars here because in these simple tests, I use them all the time and I'm not doing parallel tests
	var sdLookupKey types.NamespacedName
	var createdSD *simplegroupv0.SimpleDeployment

	BeforeEach(func() {
		By("By creating a new SimpleDeployment")
		ctx = context.Background()
		sd = &simplegroupv0.SimpleDeployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "simplegroup.mihai.domain/v0",
				Kind:       "SimpleDeployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sdName,
				Namespace: sdNamespace,
			},
			Spec: simplegroupv0.SimpleDeploymentSpec{
				Image:    sdImage,
				Replicas: func(i int32) *int32 { return &i }(sdReplicas),
				IngressInfo: simplegroupv0.IngressInfo{
					IngressControllerType: "nginx",
					IngressClassName:      "nginx",
					PublicPort:            int32(443),
					Host:                  "dev.local",
					Path:                  "/",
					RWTarget:              "",
				},
			},
		}

		Expect(k8sClient.Create(ctx, sd)).Should(Succeed())

		// Check something was created and Get it because I'll generally keep using that retrieved object in these tests
		sdLookupKey = types.NamespacedName{Name: sdName, Namespace: sdNamespace}
		createdSD = &simplegroupv0.SimpleDeployment{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, sdLookupKey, createdSD)
			if err != nil {
				return false
			}
			return true
		}, timeout, interval).Should(BeTrue())

	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, sd)).Should(Succeed())
		emptySD := &simplegroupv0.SimpleDeployment{}
		sdLookupKey := types.NamespacedName{Name: sdName, Namespace: sdNamespace}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, sdLookupKey, emptySD)
			if err != nil {
				return false
			}
			return true
		}, timeout, interval).Should(BeFalse())

		for _, t := range []client.Object{&appsv1.Deployment{}, &corev1.Service{}, &netv1.Ingress{}} {
			k8sClient.DeleteAllOf(ctx, t, client.InNamespace(sdNamespace), client.MatchingLabels{"sd-member": "true", "sd": sdName})
		}
	})

	Context("When updating SimpleDeployment Status", func() {
		It("Should update SimpleDeployment Status.Deployment when reconciled ", func() {
			// Let's make sure our Image string value was properly converted/handled.
			Expect(createdSD.Spec.Image).Should(Equal("nginx:latest"))

			// Let's check the controller logic by checking if it updates Status.Deployment correctly.
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, sdLookupKey, createdSD)
				if err != nil {
					return "", err
				}
				fmt.Printf("\ncreatedSD.GetUID()\n%+v\n\n", createdSD.GetUID())
				return createdSD.Status.Deployment, nil
			}, timeout, interval).Should(Equal(sdNamespace + "/" + sdName + "-deplo"))
		})
	})

	Context("Check owner references on objects created by the controller (example for Service) ", func() {
		It("Should find objects with an ownerReference pointing to the test SD", func() {
			svcLookupKey := types.NamespacedName{Name: sdName + "-svc", Namespace: sdNamespace}
			createdSvc := &corev1.Service{}
			// Get the created SVC
			Eventually(func() bool {
				err := k8sClient.Get(ctx, svcLookupKey, createdSvc)
				if err != nil {
					return false
				}
				fmt.Printf("\ncreatedSvc.GetOwnerReferences()\n%+v\n\n", createdSvc.GetOwnerReferences())
				return true
			}, timeout, interval).Should(BeTrue())
			// Check owner controller reference
			Expect(metav1.IsControlledBy(createdSvc, createdSD)).Should(BeTrue())
		})
	})

})
