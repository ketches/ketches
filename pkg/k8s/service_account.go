package k8s

import (
	"context"

	"github.com/pescox/go-kit/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ApplyServiceAccount(clientset *kubernetes.Clientset, sa ServiceAccount) error {
	if sa.BindingRole && len(sa.RoleName) > 0 {
		if sa.ClusterRole {
			// Apply cluster role binding
			get, err := clientset.RbacV1beta1().ClusterRoleBindings().Get(context.TODO(), sa.RoleName, metaV1.GetOptions{})
			if err != nil || len(get.Name) == 0 {
				_, err := clientset.RbacV1beta1().ClusterRoleBindings().Create(context.TODO(), &v1beta1.ClusterRoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name: sa.Name,
					},
					RoleRef: v1beta1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     string(ResourceKindClusterRoleBinding),
						Name:     sa.RoleName,
					},
					Subjects: []v1beta1.Subject{
						{
							Kind:      string(ResourceKindServiceAccount),
							Name:      sa.Name,
							Namespace: sa.Namespace,
						},
					},
				}, metaV1.CreateOptions{})
				if err != nil {
					log.Errorf("apply cluster role binding [%s] failed: %s", sa.RoleName, err.Error())
					return err
				}
			}
		} else {
			// Apply role binding
			get, err := clientset.RbacV1beta1().RoleBindings(sa.Namespace).Get(context.TODO(), sa.RoleName, metaV1.GetOptions{})
			if err != nil || len(get.Name) == 0 {
				_, err := clientset.RbacV1beta1().RoleBindings(sa.Namespace).Create(context.TODO(), &v1beta1.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      sa.Name,
						Namespace: sa.Namespace,
					},
					RoleRef: v1beta1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     string(ResourceKindRoleBinding),
						Name:     sa.RoleName,
					},
					Subjects: []v1beta1.Subject{
						{
							Kind:      string(ResourceKindServiceAccount),
							Name:      sa.Name,
							Namespace: sa.Namespace,
						},
					},
				}, metaV1.CreateOptions{})
				if err != nil {
					log.Errorf("apply cluster role binding [%s] failed.", sa.RoleName)
					return err
				}
			}
		}
	}

	get, err := clientset.CoreV1().ServiceAccounts(sa.Namespace).Get(context.TODO(), sa.Name, metaV1.GetOptions{})
	if err != nil || len(get.Name) == 0 {
		// Create services account
		_, err := clientset.CoreV1().ServiceAccounts(sa.Namespace).Create(context.TODO(), &v1.ServiceAccount{}, metaV1.CreateOptions{})
		if err != nil {
			log.Errorf("apply cluster role binding [%s] failed: %s", sa.RoleName, err.Error())
			return err
		}
	} else {
		// Update services account

	}
	return nil
}
