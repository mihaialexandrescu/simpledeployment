package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
		interval = time.Millisecond * 250
	)

	Context("When updating SimpleDeployment Status", func() {
		It("Should update SimpleDeployment Status.Deployment when reconciled ", func() {
			By("By creating a new SimpleDeployment")
			ctx := context.Background()
			sd := &simplegroupv0.SimpleDeployment{
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

			// Let's check that the created SD fields match what we passed in. Will try it for Spec.Image.
			sdLookupKey := types.NamespacedName{Name: sdName, Namespace: sdNamespace}
			createdSD := &simplegroupv0.SimpleDeployment{}

			// We'll need to retry getting this newly created SimpleDeployment, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, sdLookupKey, createdSD)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Image string value was properly converted/handled.
			Expect(createdSD.Spec.Image).Should(Equal("nginx:latest"))

			// Let's check the controller logic by checking if it updates Status.Deployment correctly.
			By("By checking the SD has a correct Status.Deployment field")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, sdLookupKey, createdSD)
				if err != nil {
					return "", err
				}
				return createdSD.Status.Deployment, nil
			}, timeout, interval).Should(Equal(sdNamespace + "/" + sdName + "-deplo"))
		})
	})
})
