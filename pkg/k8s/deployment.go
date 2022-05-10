package k8s

import (
	"context"
	"errors"
	"fmt"

	"github.com/pescox/go-kit/log"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ApplyDeployment(client *kubernetes.Clientset, deployment *models.KubeDeployment) error {
	d, err := client.AppsV1().Deployments(deployment.Namespace).Get(context.Background(), deployment.Name, v1.GetOptions{})

	ports := make([]coreV1.ContainerPort, 0)
	for _, port := range deployment.ContainerPorts {
		ports = append(ports, coreV1.ContainerPort{
			Protocol:      coreV1.ProtocolTCP,
			Name:          fmt.Sprintf("port_%d", port.Port),
			ContainerPort: port.Port,
		})
	}
	deploymentSpec := appsV1.DeploymentSpec{
		Selector: &v1.LabelSelector{
			MatchLabels: map[string]string{
				"app":     deployment.ApplicationName,
				"version": deployment.Version,
			},
		},
		Template: coreV1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: map[string]string{
					"app":     deployment.ApplicationName,
					"version": deployment.Version,
				},
			},
			Spec: coreV1.PodSpec{
				Containers: []coreV1.Container{
					{
						Name:  deployment.ApplicationName,
						Image: deployment.ContainerImage,
						Ports: ports,
					},
				},
			},
		},
	}

	var applyErr error
	if err != nil {
		d = &appsV1.Deployment{
			ObjectMeta: v1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
			},
			Spec: deploymentSpec,
		}
		_, applyErr = client.AppsV1().Deployments(deployment.Namespace).Create(context.Background(), d, v1.CreateOptions{})
	} else {
		d.Spec = deploymentSpec
		_, applyErr = client.AppsV1().Deployments(deployment.Namespace).Update(context.Background(), d, v1.UpdateOptions{})
	}
	if applyErr != nil {
		log.Errorf("apply deployment ERROR: %s", applyErr.Error())
		return errors.New("apply deployment failed")
	}
	return nil
}

func DeleteDeployment(client *kubernetes.Clientset, deploymentName, deploymentNamespace string, canary bool) error {
	deleteErr := client.AppsV1().Deployments(deploymentNamespace).Delete(context.Background(), deploymentName, v1.DeleteOptions{})
	if deleteErr != nil {
		log.Errorf("delete deployment ERROR: %s", deleteErr.Error())
		return errors.New("delete deployment failed")
	}
	return nil
}
