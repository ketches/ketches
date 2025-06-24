/*
Copyright 2025 The Ketches Authors.

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

package dynamiclister

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestGetNamespace(t *testing.T) {
	restConfig := ctrl.GetConfigOrDie()
	dynamicClient := dynamic.NewForConfigOrDie(restConfig)

	var defaultNamespace corev1.Namespace
	err := Get(dynamicClient, corev1.SchemeGroupVersion.WithResource("namespaces"), metav1.NamespaceNone, "default", &defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("--> get namespace:")
	t.Log("\t- ", defaultNamespace.Name)

	var namespaces corev1.NamespaceList
	err = List(dynamicClient, corev1.SchemeGroupVersion.WithResource("namespaces"), metav1.NamespaceNone, &namespaces)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("--> list namespaces:")
	for _, ns := range namespaces.Items {
		t.Log("\t- ", ns.Name)
	}

	var pods corev1.PodList
	err = List(dynamicClient, corev1.SchemeGroupVersion.WithResource("pods"), metav1.NamespaceAll, &pods)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("--> list pods:")
	for _, ns := range pods.Items {
		t.Log("\t- ", ns.Name)
	}
}
