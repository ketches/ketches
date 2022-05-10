package builder

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SourceBuilder struct {
	ID           string
	GitRepo      string
	GitBranch    string
	GitAccessKey string
	GitSecretKey string
	Dockerfile   string
	Context      string
}

type SourceBuilderOption struct {
	Timeout int64
}

func (b *SourceBuilder) Build(option SourceBuilderOption) (BuildResult, error) {
	taskPod := BuildTaskPod(b, option)
	// clientset := k8s.Client()

	// err := clientset.CreateTaskPod(taskPod)

	return BuildResult{string(taskPod.GetUID())}, nil
}

func BuildTaskPod(b *SourceBuilder, option SourceBuilderOption) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.ID,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "ketches",
			Containers: []corev1.Container{
				{
					Name:            "builder",
					Image:           "ketches/ketches-cli:v0.1.0",
					ImagePullPolicy: corev1.PullAlways,
					Command: []string{
						"ketches cicd",
					},
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"ketches log",
								},
							},
						},
					},
					TerminationMessagePath:   "/dev/termination-log",
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
				},
			},
		},
	}
}
