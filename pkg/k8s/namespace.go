package k8s

import (
	"context"

	"github.com/ketches/ketches/pkg/log"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetNamespace(clientset *kubernetes.Clientset, name string) (*Namespace, error) {
	kns, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return toNamespace(kns), nil
}

func ApplyNamespace(clientset *kubernetes.Clientset, ns Namespace) error {
	_, err := GetNamespace(clientset, ns.Name)
	if err != nil && apierrors.IsNotFound(err) {
		return CreateNamespace(clientset, ns)
	} else {
		return UpdateNamespace(clientset, ns)
	}
}

func toNamespace(kns *coreV1.Namespace) *Namespace {
	return &Namespace{
		Model{
			Name:        kns.Name,
			Labels:      kns.Labels,
			Annotations: kns.Annotations,
		},
	}
}

func CreateNamespace(clientset *kubernetes.Clientset, ns Namespace) error {
	kns := &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        ns.Name,
			Labels:      ns.Labels,
			Annotations: ns.Annotations,
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), kns, metaV1.CreateOptions{})
	return err
}

func UpdateNamespace(clientset *kubernetes.Clientset, ns Namespace) error {
	kns, err := clientset.CoreV1().Namespaces().Get(context.TODO(), ns.Name, metaV1.GetOptions{})
	if err != nil {
		log.Error(err)
		return err
	}

	kns.ObjectMeta.Labels = ns.Labels
	kns.ObjectMeta.Annotations = ns.Annotations

	_, err = clientset.CoreV1().Namespaces().Update(context.TODO(), kns, metaV1.UpdateOptions{})
	return err
}

func DeleteNamespace(clientset *kubernetes.Clientset, name string) error {
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metaV1.DeleteOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
