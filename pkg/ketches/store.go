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

package ketches

import (
	"log"
	"sync"

	"github.com/ketches/ketches/pkg/clusterset"
	"github.com/ketches/ketches/pkg/generated/informers/externalversions"
	"github.com/ketches/ketches/pkg/generated/listers/core/v1alpha1"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

type StoreInterface interface {
	ClusterLister() v1alpha1.ClusterLister
	SpaceLister() v1alpha1.SpaceLister
	ExtensionLister() v1alpha1.ExtensionLister
	ApplicationLister() v1alpha1.ApplicationLister
	UserLister() v1alpha1.UserLister
	RoleLister() v1alpha1.RoleLister
	AuditLister() v1alpha1.AuditLister

	Clusterset() clusterset.Clusterset
}

type store struct {
	clusterLister   v1alpha1.ClusterLister
	spaceLister     v1alpha1.SpaceLister
	extensionLister v1alpha1.ExtensionLister
	// helmRepositoryLister v1alpha1.HelmRepositoryLister
	applicationLister v1alpha1.ApplicationLister
	userLister        v1alpha1.UserLister
	roleLister        v1alpha1.RoleLister
	auditLister       v1alpha1.AuditLister

	clusterset clusterset.Clusterset
}

func (s *store) ClusterLister() v1alpha1.ClusterLister {
	return s.clusterLister
}

func (s *store) SpaceLister() v1alpha1.SpaceLister {
	return s.spaceLister
}

func (s *store) ExtensionLister() v1alpha1.ExtensionLister {
	return s.extensionLister
}

func (s *store) ApplicationLister() v1alpha1.ApplicationLister {
	return s.applicationLister
}

func (s *store) UserLister() v1alpha1.UserLister {
	return s.userLister
}

func (s *store) RoleLister() v1alpha1.RoleLister {
	return s.roleLister
}

func (s *store) AuditLister() v1alpha1.AuditLister {
	return s.auditLister
}

func (s *store) Clusterset() clusterset.Clusterset {
	return s.clusterset
}

var once sync.Once
var cachedStore StoreInterface

func Store() StoreInterface {
	once.Do(loadStore)

	return cachedStore
}

func loadStore() {
	ketchesInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(Client(), 0)

	cluster := ketchesInformerFactory.Core().V1alpha1().Clusters()
	clusterInformer := cluster.Informer()
	space := ketchesInformerFactory.Core().V1alpha1().Spaces()
	spaceInformer := space.Informer()
	extension := ketchesInformerFactory.Core().V1alpha1().Extensions()
	extensionInformer := extension.Informer()
	application := ketchesInformerFactory.Core().V1alpha1().Applications()
	applicationInformer := application.Informer()
	user := ketchesInformerFactory.Core().V1alpha1().Users()
	userInformer := user.Informer()
	role := ketchesInformerFactory.Core().V1alpha1().Roles()
	roleInformer := role.Informer()
	audit := ketchesInformerFactory.Core().V1alpha1().Audits()
	auditInformer := audit.Informer()

	cs := clusterset.NewClusterset()
	clusterInformer.AddEventHandler(clusterEventHandler(cs))

	ketchesInformerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		clusterInformer,
		spaceInformer,
		extensionInformer,
		applicationInformer,
		userInformer,
		roleInformer,
		auditInformer,
	}
	var wg sync.WaitGroup
	wg.Add(len(sharedInformers))
	for _, si := range sharedInformers {
		go func(si cache.SharedInformer) {
			if !cache.WaitForCacheSync(wait.NeverStop, si.HasSynced) {
				panic("timed out waiting for caches to sync")
			}
			wg.Done()
		}(si)
	}
	wg.Wait()

	cachedStore = &store{
		clusterLister:     cluster.Lister(),
		extensionLister:   extension.Lister(),
		spaceLister:       space.Lister(),
		applicationLister: application.Lister(),
		userLister:        user.Lister(),
		roleLister:        role.Lister(),
		auditLister:       audit.Lister(),

		clusterset: cs,
	}
}

var clusterEventHandler = func(cs clusterset.Clusterset) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c := obj.(*corev1alpha1.Cluster)

			cluster := clusterset.NewCluster(c)
			if cluster != nil {
				cs.Set(c.Name, cluster)
				log.Printf("cluster %s loaded", c.Name)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newc := newObj.(*corev1alpha1.Cluster)
			oldc := oldObj.(*corev1alpha1.Cluster)
			if newc.ResourceVersion == oldc.ResourceVersion {
				return
			}

			if newc.Spec.KubeConfig == oldc.Spec.KubeConfig {
				return
			}

			cluster, ok := cs.Cluster(newc.Name)
			if !ok {
				cluster = clusterset.NewCluster(newc)
			}
			cluster.Reset()
			if cluster != nil {
				cs.Set(newc.Name, cluster)
				log.Printf("cluster %s resynced", newc.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			c := obj.(*corev1alpha1.Cluster)
			cs.Forget(c.Name)
			log.Printf("cluster %s discarded", c.Name)
		},
	}
}
