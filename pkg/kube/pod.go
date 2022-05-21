package kube

// import (
// 	"context"
// 	"fmt"
// 	"path"

// 	"github.com/ketches/ketches/pkg/log"
// 	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes"
// )

// func CreateTaskPod(clientset *kubernetes.Clientset, p Pod) error {
// 	resourceSuffix := p.Name
// 	if len(p.MountFiles) > 0 {
// 		configMapData := map[string]string{}
// 		for _, mf := range pod.MountFiles {
// 			configMapData[mf.Id] = mf.Content
// 		}
// 		clientset.CoreV1().ConfigMaps(p.Namespace).Create(context.TODO(), &corev1.ConfigMap{
// 			ObjectMeta: metaV1.ObjectMeta{
// 				Name:      fmt.Sprintf("cf-%s", resourceSuffix),
// 				Namespace: p.Namespace,
// 			},
// 			Data: configMapData,
// 		}, metaV1.CreateOptions{})
// 	}

// 	env := make([]coreV1.EnvVar, 0)
// 	for k, v := range pod.EnvironmentVariables {
// 		env = append(env, coreV1.EnvVar{
// 			Name:  k,
// 			Value: v,
// 		})
// 	}

// 	volumeMounts := make([]coreV1.VolumeMount, 0)
// 	for _, file := range pod.MountFiles {
// 		volumeMounts = append(volumeMounts, coreV1.VolumeMount{
// 			Name:      file.Id,
// 			MountPath: file.AbsolutePath,
// 			SubPath:   path.Dir(file.AbsolutePath), // todo...
// 		})
// 	}

// 	p := &coreV1.Pod{
// 		ObjectMeta: metaV1.ObjectMeta{
// 			Name:      pod.Name,
// 			Namespace: pod.Namespace,
// 			Labels: map[string]string{
// 				"task-id": pod.TaskId,
// 			},
// 		},
// 		Spec: coreV1.PodSpec{
// 			//ServiceAccountName: pod.ServiceAccount,
// 			Containers: []coreV1.Container{
// 				{
// 					Name:            pod.Name,
// 					Image:           pod.Image,
// 					ImagePullPolicy: "Always",
// 					Env:             env,
// 					Command:         pod.Command,
// 					VolumeMounts:    volumeMounts,
// 				},
// 			},
// 			Volumes: []coreV1.Volume{
// 				{
// 					Name: "heimdallr-ci-vol",
// 					VolumeSource: coreV1.VolumeSource{
// 						ConfigMap: &coreV1.ConfigMapVolumeSource{
// 							LocalObjectReference: coreV1.LocalObjectReference{
// 								Name: "heimdallr-ci-configmap",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			RestartPolicy: coreV1.RestartPolicyOnFailure,
// 		},
// 	}

// 	if pod.ExitGracefully {
// 		p.Spec.Containers[0].Lifecycle = &coreV1.Lifecycle{
// 			PreStop: &coreV1.Handler{
// 				Exec: &coreV1.ExecAction{
// 					Command: pod.ExitCommand,
// 				},
// 			},
// 		}
// 	}

// 	if pod.DockerRequired {
// 		p.Spec.Containers[0].Env = append(p.Spec.Containers[0].Env, coreV1.EnvVar{
// 			Name:  "DOCKER_HOST",
// 			Value: "tcp://localhost:2375",
// 		})
// 		p.Spec.Containers = append(p.Spec.Containers, coreV1.Container{
// 			Name:  "dind-daemon", // docker in docker
// 			Image: "docker:18-dind",
// 			SecurityContext: &coreV1.SecurityContext{
// 				Privileged: util.BoolPtr(true),
// 			},
// 		})
// 	}

// 	_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(context.TODO(), p, metaV1.CreateOptions{})
// 	return err
// }

// func GetPod(kubeClient *kubernetes.Clientset, namespace, pod string) models.KubePod {
// 	var res models.KubePod
// 	p, err := kubeClient.CoreV1().Pods(namespace).Get(context.Background(), pod, metaV1.GetOptions{})
// 	if err != nil {
// 		log.Errorf("get pod failed: %s", err)
// 		return res
// 	}
// 	return toKubePod(*p)
// }

// func toKubePod(pod coreV1.Pod) models.KubePod {
// 	var image string
// 	containers := pod.Spec.Containers
// 	for _, container := range containers {
// 		if container.Name == pod.Name {
// 			image = container.Image
// 		}
// 	}
// 	return Pod{
// 		Name:  pod.Name,
// 		Image: image,
// 	}
// }
