/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package initializer

import (
	"context"
	"github.com/ketches/ketches/api/spec"
	"slices"

	"github.com/fatih/color"
	"github.com/ketches/ketches/api/core/v1alpha1"
	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/global"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/pkg/kube"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
)

func InitializePlatform() error {
	// builtin namespace
	_, err := kube.Client().CoreV1().Namespaces().Get(context.Background(), global.BuiltinNamespace, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err := kube.Client().CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   global.BuiltinNamespace,
				Labels: corev1alpha1.BuiltinResourceLabels(),
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	// builtin admin user
	_, err = ketches.Store().UserLister().Get("admin")
	if err != nil && errors.IsNotFound(err) {
		_, err := ketches.Client().CoreV1alpha1().Users().Create(context.Background(), builtinAdminUser, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		color.Cyan("Initialized admin user.\nAccount: %s\nPassword: %s\nPlease modify the password after first login.\n", builtinAdminUser.Name, adminPassword)
	}

	// builtin roles
	roles, err := ketches.Store().RoleLister().List(labels.Everything())
	if err != nil {
		return err
	}
	for _, role := range builtinRoles {
		if !slices.ContainsFunc(roles, func(r *v1alpha1.Role) bool {
			return r.Name == role.Name
		}) {
			_, err = ketches.Client().CoreV1alpha1().Roles().Create(context.Background(), &role, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var adminPassword = rand.String(16)

var builtinAdminUser = &v1alpha1.User{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "admin",
		Labels: corev1alpha1.BuiltinResourceLabels(),
	},
	Spec: v1alpha1.UserSpec{
		ViewSpec: spec.ViewSpec{
			DisplayName: "Admin",
			Description: "Admin user, operator of the platform",
		},
		Builtin: true,
		Email:   "admin@" + rand.String(12) + ".ketches.io",
		PasswordHash: func() string {
			hashPassword, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			return string(hashPassword)
		}(),
		MustResetPassword: true,
	},
}

// builtinRoles can not be deleted and modified by normal users
var builtinRoles = map[string]v1alpha1.Role{
	"platform-manager": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "platform-manager",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Platform Manager",
				Description: "Can manage all resources in the platform",
			},
		},
	},
	"platform-developer": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "platform-developer",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Platform Developer",
				Description: "Platform developer can manage resources around applications",
			},
		},
	},
	"platform-viewer": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "platform-viewer",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Platform Viewer",
				Description: "Platform viewer wait to be invited to a space",
			},
		},
	},
	"space-owner": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "space-manager",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Space Owner",
				Description: "Owns the space and can manage all resources in the space",
			},
		},
	},
	"space-maintainer": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "space-maintainer",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Space Maintainer",
				Description: "Can manage all resources in the space",
			},
		},
	},
	"space-viewer": {
		ObjectMeta: metav1.ObjectMeta{
			Name:   "space-viewer",
			Labels: corev1alpha1.BuiltinResourceLabels(),
		},
		Spec: v1alpha1.RoleSpec{
			Builtin: true,
			ViewSpec: spec.ViewSpec{
				DisplayName: "Space Viewer",
				Description: "Can view all resources in the space",
			},
		},
	},
}

func hasRole(roles []*v1alpha1.Role, role v1alpha1.Role) bool {
	for _, r := range roles {
		if r.Name == role.Name {
			return true
		}
	}
	return false
}
