package core

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxNamespaceBaseLength = 57
	randomSuffixLength     = 5
)

func GenerateNamespaceName(projectSlug, envSlug string) string {
	base := fmt.Sprintf("%s-%s", projectSlug, envSlug)

	if len(base) > maxNamespaceBaseLength {
		base = base[:maxNamespaceBaseLength]
	}

	suffix := generateRandomString(randomSuffixLength)

	return fmt.Sprintf("%s-%s", base, suffix)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func CreateNamespace(ctx context.Context, clusterID, namespaceName string, envCtx *models.EnvContext) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	labels := map[string]string{
		kube.LabelManagedBy: "true",
	}
	if envCtx != nil {
		labels[kube.LabelEnvID] = envCtx.Env.ID
		labels[kube.LabelEnvSlug] = envCtx.Env.Slug
		labels[kube.LabelProjectID] = envCtx.Project.ID
		labels[kube.LabelProjectSlug] = envCtx.Project.Slug
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: labels,
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	}

	return nil
}

func DeleteNamespace(ctx context.Context, clusterID, namespaceName string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	err = client.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}

	return nil
}
