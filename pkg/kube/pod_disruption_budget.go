package kube

import (
	"context"

	"k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func ApplyPodDisruptionBudget(clientset *kubernetes.Clientset, pdb PodDisruptionBudget) error {
	kpdb, err := clientset.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace).Get(context.Background(), pdb.Name, metav1.GetOptions{})

	kpdbSpec := v1beta1.PodDisruptionBudgetSpec{
		MinAvailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: pdb.MinAvailablePodReplicas,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: pdb.SelectorMatchLabels,
		},
	}

	if err != nil && apierrors.IsNotFound(err) {
		kpdb = &v1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pdb.Name,
				Namespace: pdb.Namespace,
			},
			Spec: kpdbSpec,
		}
		_, err := clientset.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace).Create(context.Background(), kpdb, metav1.CreateOptions{})
		return err
	}
	if kpdb == nil {
		kpdb = &v1beta1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pdb.Name,
				Namespace: pdb.Namespace,
			},
		}
	}
	kpdb.Spec = kpdbSpec
	_, err = clientset.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace).Update(context.Background(), kpdb, metav1.UpdateOptions{})
	return err
}

func DeletePodDisruptionBudget(clientset *kubernetes.Clientset, name, namespace string) error {
	return clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}
