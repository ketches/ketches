package kube

import (
	"context"
	"errors"

	"github.com/pescox/go-kit/log"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type NginxIngress struct {
}

func (ingress NginxIngress) ApplyGateway(client *kubernetes.Clientset, gateway ApplyGatewayModel) error {
	ing, err := client.ExtensionsV1beta1().Ingresses(gateway.Namespace).Get(context.Background(), gateway.Name, v1.GetOptions{})

	hosts := make([]string, 0)
	rules := make([]v1beta1.IngressRule, 0)
	for _, host := range gateway.Hosts {
		hosts = append(hosts, host.Host)

		paths := make([]v1beta1.HTTPIngressPath, 0)

		for _, service := range host.Services {
			paths = append(paths, v1beta1.HTTPIngressPath{
				Path: service.Path,
				Backend: v1beta1.IngressBackend{
					ServiceName: service.Name,
					ServicePort: intstr.Parse(service.Port),
				},
			})
		}

		rules = append(rules, v1beta1.IngressRule{
			Host: host.Host,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		})
	}

	ingSpec := v1beta1.IngressSpec{
		TLS: []v1beta1.IngressTLS{
			{
				Hosts:      hosts,
				SecretName: "tls-secret",
			},
		},
		Rules: rules,
	}
	if err != nil {
		// create
		ing = &v1beta1.Ingress{
			ObjectMeta: v1.ObjectMeta{
				Name:        gateway.Name,
				Namespace:   gateway.Namespace,
				Labels:      gateway.Labels,
				Annotations: gateway.Annotations,
			},
			Spec: ingSpec,
		}

		_, err := client.ExtensionsV1beta1().Ingresses(gateway.Namespace).Create(context.Background(), ing, v1.CreateOptions{})
		if err != nil {
			log.ErrorF("apply ingress error: %s", err.Error())
			return errors.New("apply ingress failed")
		}
	} else {
		// update
		ing.Labels = gateway.Labels
		ing.Annotations = gateway.Annotations
		ing.Spec = ingSpec

		_, err := client.ExtensionsV1beta1().Ingresses(gateway.Namespace).Update(context.Background(), ing, v1.UpdateOptions{})
		if err != nil {
			log.ErrorF("apply ingress error: %s", err.Error())
			return errors.New("apply ingress failed")
		}
	}
	return nil
}

func (ingress NginxIngress) DeleteGateway(client *kubernetes.Clientset, gateway DeleteGatewayModel) error {
	err := client.ExtensionsV1beta1().Ingresses(gateway.Namespace).Delete(context.Background(), gateway.Name, v1.DeleteOptions{})
	if err != nil {
		log.ErrorF("delete ingress error: %s", err.Error())
		return errors.New("delete ingress failed")
	}
	return nil
}
