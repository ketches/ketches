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
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/ketches/ketches/pkg/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Applications returns a ApplicationInformer.
	Applications() ApplicationInformer
	// ApplicationGroups returns a ApplicationGroupInformer.
	ApplicationGroups() ApplicationGroupInformer
	// Audits returns a AuditInformer.
	Audits() AuditInformer
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// Extensions returns a ExtensionInformer.
	Extensions() ExtensionInformer
	// HelmRepositories returns a HelmRepositoryInformer.
	HelmRepositories() HelmRepositoryInformer
	// Roles returns a RoleInformer.
	Roles() RoleInformer
	// Spaces returns a SpaceInformer.
	Spaces() SpaceInformer
	// Users returns a UserInformer.
	Users() UserInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Applications returns a ApplicationInformer.
func (v *version) Applications() ApplicationInformer {
	return &applicationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ApplicationGroups returns a ApplicationGroupInformer.
func (v *version) ApplicationGroups() ApplicationGroupInformer {
	return &applicationGroupInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Audits returns a AuditInformer.
func (v *version) Audits() AuditInformer {
	return &auditInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Clusters returns a ClusterInformer.
func (v *version) Clusters() ClusterInformer {
	return &clusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Extensions returns a ExtensionInformer.
func (v *version) Extensions() ExtensionInformer {
	return &extensionInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// HelmRepositories returns a HelmRepositoryInformer.
func (v *version) HelmRepositories() HelmRepositoryInformer {
	return &helmRepositoryInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Roles returns a RoleInformer.
func (v *version) Roles() RoleInformer {
	return &roleInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Spaces returns a SpaceInformer.
func (v *version) Spaces() SpaceInformer {
	return &spaceInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}