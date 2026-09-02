package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxNamespaceLength  = 63
	namespaceHashLength = 10
)

var invalidNamespaceChars = regexp.MustCompile(`[^a-z0-9-]+`)

func GenerateNamespaceName(projectSlug, envSlug string) string {
	base := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%s-%s", projectSlug, envSlug)))
	base = invalidNamespaceChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	base = strings.Join(strings.FieldsFunc(base, func(r rune) bool {
		return r == '-'
	}), "-")

	if len(base) > maxNamespaceLength {
		sum := sha256.Sum256([]byte(base))
		suffix := hex.EncodeToString(sum[:])[:namespaceHashLength]
		prefixLength := maxNamespaceLength - namespaceHashLength - 1
		base = strings.Trim(base[:prefixLength], "-") + "-" + suffix
	}

	return base
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

	existing, err := client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err == nil {
		return validateNamespaceOwnership(existing, labels)
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			existing, getErr := client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			return validateNamespaceOwnership(existing, labels)
		}
		return err
	}

	if err := CreateDefaultResourceQuota(ctx, clusterID, namespaceName); err != nil {
		return err
	}

	return nil
}

func validateNamespaceOwnership(namespace *corev1.Namespace, expectedLabels map[string]string) error {
	if namespace == nil {
		return app.NewErrorf("namespace is required")
	}
	expectedEnvID := expectedLabels[kube.LabelEnvID]
	actualEnvID := namespace.Labels[kube.LabelEnvID]
	if namespace.Labels[kube.LabelManagedBy] != "true" || expectedEnvID == "" || actualEnvID != expectedEnvID {
		return app.NewErrorf(
			"namespace %q is not owned by environment %q",
			namespace.Name,
			expectedEnvID,
		)
	}
	if expectedProjectID := expectedLabels[kube.LabelProjectID]; namespace.Labels[kube.LabelProjectID] != expectedProjectID {
		return app.NewErrorf("namespace %q is not owned by project %q", namespace.Name, expectedProjectID)
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
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func NamespaceExists(ctx context.Context, clusterID, namespaceName string) (bool, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return false, err
	}

	_, err = client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
