/*
Copyright 2023 The Ketches Authors.

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

package kube

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"testing"
)

func TestListNamespaces(t *testing.T) {
	var defaultNamespace corev1.Namespace
	err := Store().NamespaceLister().Get("default").ToObject(&defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	var namespaceList corev1.NamespaceList
	err = Store().NamespaceLister().List(labels.Everything()).ToObjectList(&namespaceList)

	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range namespaceList.Items {
		t.Log("\t- ", ns.Name, ns.Status.Phase)
	}
}
