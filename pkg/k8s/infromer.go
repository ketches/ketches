package k8s

import (
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
)

func SharedInformer() informers.SharedInformerFactory {
	return informers.NewSharedInformerFactoryWithOptions(Client(), 0, informers.WithNamespace("default"), informers.WithTweakListOptions(func(options *informers.ListOptions) {
		options.LabelSelector = "app=ketches"
	}))
}

func PodInformer() corev1.PodInformer {
	return SharedInformer().Core().V1().Pods()
}
