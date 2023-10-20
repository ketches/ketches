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

package handler

// import (
// 	"github.com/gin-gonic/gin"
// 	"github.com/ketches/ketches/pkg/http"
// 	"github.com/ketches/ketches/pkg/kube"
// 	"k8s.io/apimachinery/pkg/labels"
// )
//
// func ListClusters(c *gin.Context) {
// 	ret, err := kube.Store().ClusterLister.List(labels.Everything())
// 	if err != nil {
// 		http.Error(c, err)
// 	}
//
// 	http.Success(c, ret)
// }
//
// func GetCluster(c *gin.Context) {
// 	clusterName := c.Param("name")
// 	ret, err := kube.Store().ClusterLister.Get(clusterName)
// 	if err != nil {
// 		http.Error(c, err)
// 	}
//
// 	http.Success(c, ret)
// }
