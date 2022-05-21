package kube

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
)

func SharedInformer() informers.SharedInformerFactory {
	return informers.NewSharedInformerFactoryWithOptions(
		Clientset(),
		time.Hour*8,
		// informers.WithNamespace("default"),
		informers.WithTweakListOptions(func(lo *v1.ListOptions) {
			lo.LabelSelector = "app=ketches"
		}),
	)
}

func PodInformer() corev1.PodInformer {
	return SharedInformer().Core().V1().Pods()
}
