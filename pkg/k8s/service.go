package k8s

import (
	"context"
	"fmt"

	"github.com/pescox/go-kit/exp"
	coreV1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func ApplyService(clientset *kubernetes.Clientset, s *Service) error {
	ks, err := clientset.CoreV1().Services(s.Namespace).Get(context.Background(), s.Name, v1.GetOptions{})

	ports := make([]coreV1.ServicePort, 0)
	for _, p := range s.Ports {
		ports = append(ports, coreV1.ServicePort{
			Protocol:   corev1.Protocol(exp.If(len(p.Protocol) == 0, "TCP", p.Protocol)),
			Name:       exp.If(len(p.Name) == 0, fmt.Sprint("p_%d", p.Port), p.Name),
			Port:       p.Port,
			TargetPort: intstr.Parse(p.TargetPort),
			NodePort:   p.NodePort,
		})
	}
	ksSpec := coreV1.ServiceSpec{
		Selector: s.SelectorMatchLabels,
		Ports:    ports,
		Type:     coreV1.ServiceType(exp.If(len(s.Type) == 0, "NodePort", string(s.Type))),
	}

	if err != nil && apierrors.IsNotFound(err) {
		ks = &coreV1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
			Spec: ksSpec,
		}
		_, err = clientset.CoreV1().Services(s.Namespace).Create(context.Background(), ks, v1.CreateOptions{})
		return err
	}
	if ks == nil {
		ks = &coreV1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
		}
	}
	ks.Spec = ksSpec
	_, err = clientset.CoreV1().Services(s.Namespace).Update(context.Background(), ks, v1.UpdateOptions{})
	return err
}

func DeleteService(clientset *kubernetes.Clientset, name, namespace string) error {
	return clientset.CoreV1().Services(namespace).Delete(context.Background(), name, v1.DeleteOptions{})
}

func GetServiceLoadBalancer(clientset *kubernetes.Clientset, name, namespace string) (string, error) {
	s, err := client.CoreV1().Services(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err == nil && s.Spec.Type == coreV1.ServiceTypeLoadBalancer {
		return exp.If(len(s.Status.LoadBalancer.Ingress[0].Hostname) > 0, s.Status.LoadBalancer.Ingress[0].Hostname, s.Status.LoadBalancer.Ingress[0].IP), nil
	}

	return "", err
}
