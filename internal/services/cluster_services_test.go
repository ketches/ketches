package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildClusterServiceResponses(t *testing.T) {
	serviceItems := []corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "prometheus"},
			Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       9090,
					TargetPort: intstr.FromInt32(9090),
				},
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       10901,
					TargetPort: intstr.FromString("grpc"),
					NodePort:   31090,
				},
			}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "grafana"},
			Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       3000,
					TargetPort: intstr.FromInt32(3000),
				},
			}},
		},
	}

	responses := buildClusterServiceResponses(serviceItems)

	require.Len(t, responses, 2)

	assert.Equal(t, "prometheus", responses[0].Name)
	require.Len(t, responses[0].Ports, 2)
	assert.Equal(t, "http", responses[0].Ports[0].Name)
	assert.Equal(t, "TCP", responses[0].Ports[0].Protocol)
	assert.Equal(t, int32(9090), responses[0].Ports[0].Port)
	assert.Equal(t, "9090", responses[0].Ports[0].TargetPort)
	assert.Equal(t, int32(0), responses[0].Ports[0].NodePort)

	assert.Equal(t, "grpc", responses[0].Ports[1].Name)
	assert.Equal(t, int32(10901), responses[0].Ports[1].Port)
	assert.Equal(t, "grpc", responses[0].Ports[1].TargetPort)
	assert.Equal(t, int32(31090), responses[0].Ports[1].NodePort)

	assert.Equal(t, "grafana", responses[1].Name)
	require.Len(t, responses[1].Ports, 1)
	assert.Equal(t, int32(3000), responses[1].Ports[0].Port)
	assert.Equal(t, "3000", responses[1].Ports[0].TargetPort)
}
