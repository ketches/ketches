package kube

import (
	"context"
	"fmt"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	os.Setenv("KUBE_CONFIG", "/home/dp/.kube/config")
}

func TestAllGroupVersionResources(t *testing.T) {
	gvrs := AllGroupVersionResources()
	for _, gvk := range gvrs {
		fmt.Println(gvk.Group, gvk.Version, gvk.Resource)
	}
}

func TestAllGroupVersionKinds(t *testing.T) {
	// AllGroupVersionKinds()
	gvks := AllGroupVersionKinds()
	for _, gvk := range gvks {
		fmt.Println(gvk.Group, gvk.Version, gvk.Kind)
	}
}

func TestDynamic(t *testing.T) {
	svc, err := DynamicClient().Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}).Namespace("default").Get(context.TODO(), "nginx", metav1.GetOptions{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(svc.GetKind(), svc.GetNamespace(), svc.GetName(), svc.GetLabels())
	}

	deploy, err := DynamicClient().Resource(schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
		// Resource: "Deployment", // can't use this because it required a plural
	}).Namespace("default").Get(context.TODO(), "nginx", metav1.GetOptions{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(deploy.GetKind(), deploy.GetNamespace(), deploy.GetName(), deploy.GetLabels())
	}
}
