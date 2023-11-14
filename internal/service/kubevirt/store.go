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

package kubevirt

// import (
// 	"sync"

// 	"github.com/ketches/ketches/pkg/kube/dynamiclister"
// 	"k8s.io/client-go/dynamic"
// 	"k8s.io/client-go/dynamic/dynamicinformer"
// )

// type StoreInterface interface {
// 	VirtualMachineLister() dynamiclister.GenericLister
// 	VirtualMachineInstanceLister() dynamiclister.GenericLister
// }

// type store struct {
// 	virtualMachineLister        dynamiclister.GenericLister
// 	virtualMachinInstanceLister dynamiclister.GenericLister
// }

// var cachedStore StoreInterface
// var lock sync.Mutex

// func Store(dynamicClient) StoreInterface {
// 	if cachedStore == nil {
// 		b := lock.TryLock()
// 		if b {
// 			defer lock.Unlock()
// 			loadCachedStore(dynamicClient)
// 		} else {
// 			lock.Lock()
// 			defer lock.Unlock()
// 			if cachedStore == nil {
// 				loadCachedStore(dynamicClient)
// 			}
// 		}
// 	}

// 	return cachedStore
// }

// func loadCachedStore(dynamicClient dynamic.Interface) {
// 	dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

// 	_store.virtualMachineLister = dynamiclister.NewGenericLister()

// 	cachedStore = &store{}
// }
