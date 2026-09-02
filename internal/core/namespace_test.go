package core

import (
	"context"
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGenerateNamespaceNameAddsStableHashWhenTruncated(t *testing.T) {
	prefix := strings.Repeat("long-project-", 5)
	first := GenerateNamespaceName(prefix, "production-a")
	second := GenerateNamespaceName(prefix, "production-b")

	if len(first) > maxNamespaceLength || len(second) > maxNamespaceLength {
		t.Fatalf("generated namespaces exceed %d characters: %q, %q", maxNamespaceLength, first, second)
	}
	if first == second {
		t.Fatalf("truncated namespaces collided: %q", first)
	}
	if first != GenerateNamespaceName(prefix, "production-a") {
		t.Fatalf("namespace generation is not stable: %q", first)
	}
}

func TestApplyNamespacePreservesClusterMetadata(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "project-prod",
			ResourceVersion: "7",
			Labels: map[string]string{
				kube.LabelManagedBy:         "true",
				kube.LabelEnvID:             "env-1",
				kube.LabelProjectID:         "project-1",
				"platform.example/retained": "true",
			},
			Annotations: map[string]string{"platform.example/note": "retained"},
		},
		Spec: corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes}},
	})
	desired := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "project-prod",
		Labels: map[string]string{
			kube.LabelManagedBy:   "true",
			kube.LabelEnvID:       "env-1",
			kube.LabelEnvSlug:     "production",
			kube.LabelProjectID:   "project-1",
			kube.LabelProjectSlug: "project",
		},
	}}

	if err := ApplyResource(context.Background(), client, desired); err != nil {
		t.Fatalf("ApplyResource(Namespace): %v", err)
	}
	updated, err := client.CoreV1().Namespaces().Get(context.Background(), desired.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Labels["platform.example/retained"] != "true" || updated.Annotations["platform.example/note"] != "retained" {
		t.Fatalf("cluster metadata was not retained: %#v", updated.ObjectMeta)
	}
	if updated.Labels[kube.LabelEnvSlug] != "production" {
		t.Fatalf("desired ownership labels were not applied: %#v", updated.Labels)
	}
	if len(updated.Spec.Finalizers) != 1 || updated.Spec.Finalizers[0] != corev1.FinalizerKubernetes {
		t.Fatalf("namespace finalizers were not retained: %#v", updated.Spec.Finalizers)
	}
}

func TestApplyNamespaceRejectsDifferentOwner(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "shared",
		Labels: map[string]string{
			kube.LabelManagedBy: "true",
			kube.LabelEnvID:     "other-env",
			kube.LabelProjectID: "project-1",
		},
	}})
	desired := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "shared",
		Labels: map[string]string{
			kube.LabelManagedBy: "true",
			kube.LabelEnvID:     "env-1",
			kube.LabelProjectID: "project-1",
		},
	}}

	if err := ApplyResource(context.Background(), client, desired); err == nil {
		t.Fatal("ApplyResource(Namespace) accepted a namespace owned by another environment")
	}
}
