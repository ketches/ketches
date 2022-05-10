package k8s

import (
	"context"

	"github.com/ketches/ketches/pkg/cast"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateHorizontalPodAutoscaler(clientset *kubernetes.Clientset, hpa HorizontalPodAutoscaler) error {
	khpa, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Get(context.Background(), hpa.Name, v1.GetOptions{})

	khpaSpec := autoscalingv2.HorizontalPodAutoscalerSpec{
		MaxReplicas: hpa.MaxReplicas,
		MinReplicas: &hpa.MinReplicas,
		ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       string(hpa.ScaleTargetKind),
			Name:       hpa.ScaleTargetName,
		},
		Metrics: []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &hpa.TargetCPUAverageUtilization,
					},
				},
			},
		},
		Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
			ScaleDown: &autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: cast.Ptr(int32(10)),
			},
		},
	}

	if err != nil && apierrors.IsNotFound(err) {
		khpa = &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hpa.Name,
				Namespace: hpa.Namespace,
			},
			Spec: khpaSpec,
		}
		_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Create(context.Background(), khpa, v1.CreateOptions{})
		return err
	}
	if khpa == nil {
		khpa = &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hpa.Name,
				Namespace: hpa.Namespace,
			},
		}
	}
	khpa.Spec = khpaSpec
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Update(context.Background(), khpa, v1.UpdateOptions{})
	return err
}

func DeleteHorizontalPodAutoscaler(clientset *kubernetes.Clientset, name, namespace string) error {
	return clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(context.Background(), name, v1.DeleteOptions{})
}
