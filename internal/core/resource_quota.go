package core

import (
	"context"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ResourceQuotaName = "ketches-resource-quota"

func BuildResourceQuota(namespace string, quota *models.UpdateResourceQuotaRequest) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceQuotaName,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:          resource.MustParse(quota.CPURequest),
				corev1.ResourceLimitsCPU:    resource.MustParse(quota.CPULimit),
				corev1.ResourceMemory:       resource.MustParse(quota.MemoryRequest),
				corev1.ResourceLimitsMemory: resource.MustParse(quota.MemoryLimit),
				corev1.ResourcePods:         resource.MustParse(quota.Pods),
			},
		},
	}
}

func buildDefaultResourceQuota(namespace string) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceQuotaName,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:          resource.MustParse("2"),
				corev1.ResourceLimitsCPU:    resource.MustParse("4"),
				corev1.ResourceMemory:       resource.MustParse("4Gi"),
				corev1.ResourceLimitsMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:         resource.MustParse("50"),
			},
		},
	}
}

func CreateDefaultResourceQuota(ctx context.Context, clusterID, namespace string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	rq := buildDefaultResourceQuota(namespace)
	_, err = client.CoreV1().ResourceQuotas(namespace).Create(ctx, rq, metav1.CreateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func ApplyResourceQuota(ctx context.Context, clusterID, namespace string, quota *models.UpdateResourceQuotaRequest) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	rq := BuildResourceQuota(namespace, quota)

	existing, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, ResourceQuotaName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = client.CoreV1().ResourceQuotas(namespace).Create(ctx, rq, metav1.CreateOptions{})
			return err
		}
		return err
	}

	rq.ResourceVersion = existing.ResourceVersion
	_, err = client.CoreV1().ResourceQuotas(namespace).Update(ctx, rq, metav1.UpdateOptions{})
	return err
}

func GetResourceQuota(ctx context.Context, clusterID, namespace string) (*models.ResourceQuotaResponse, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}

	existing, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, ResourceQuotaName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &models.ResourceQuotaResponse{
				CPURequest:    "2",
				CPULimit:      "4",
				MemoryRequest: "4Gi",
				MemoryLimit:   "8Gi",
				Pods:          "50",
			}, nil
		}
		return nil, err
	}

	hard := existing.Spec.Hard
	cpuLimit := hard[corev1.ResourceLimitsCPU]
	memoryLimit := hard[corev1.ResourceLimitsMemory]
	pods := hard[corev1.ResourcePods]

	return &models.ResourceQuotaResponse{
		CPURequest:    hard.Cpu().String(),
		CPULimit:      cpuLimit.String(),
		MemoryRequest: hard.Memory().String(),
		MemoryLimit:   memoryLimit.String(),
		Pods:          pods.String(),
	}, nil
}
