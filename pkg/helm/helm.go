package helm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/ketches/ketches/pkg/kube"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/apimachinery/pkg/api/meta"
)

var conf *action.Configuration

func Configuration(namespace string) *action.Configuration {
	actionConfig := new(action.Configuration)
	k8sConfig := kube.RestConfig()
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.APIServer = &k8sConfig.Host
	kubeConfig.BearerToken = &k8sConfig.BearerToken
	kubeConfig.CAFile = &k8sConfig.CAFile
	kubeConfig.Namespace = &namespace
	if err := actionConfig.Init(kubeConfig, namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return nil
	}
	return actionConfig
}

type Resource struct {
	ApiVersion        string
	Kind              string
	Name              string
	Namespace         string
	CreationTimestamp time.Time
	Labels            map[string]string
	Annotations       map[string]string
}

func ResourcesFromManifest(name, namespace string) ([]Resource, error) {
	actionConfig := Configuration(namespace)
	release, err := action.NewGet(actionConfig).Run(name)
	if err != nil {
		return nil, err
	}

	objs, err := ToObjects(bytes.NewReader([]byte(release.Manifest)))
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, obj := range objs {
		o, err := meta.Accessor(obj)

		if err != nil {
			return nil, err
		}
		r := Resource{
			Name:              o.GetName(),
			Namespace:         o.GetNamespace(),
			CreationTimestamp: o.GetCreationTimestamp().Time,
			Labels:            o.GetLabels(),
			Annotations:       o.GetAnnotations(),
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		r.ApiVersion, r.Kind = gvk.ToAPIVersionAndKind()
		resources = append(resources, r)
	}

	return resources, nil
}

func ToObjects(in io.Reader) ([]runtime.Object, error) {
	var result []runtime.Object
	reader := yaml.NewYAMLReader(bufio.NewReaderSize(in, 4096))
	for {
		raw, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		bytes, err := yaml.ToJSON(raw)
		if err != nil {
			return nil, err
		}
		check := map[string]interface{}{}
		if err := json.Unmarshal(bytes, &check); err != nil {
			return nil, err
		} else if len(check) == 0 {
			continue
		}

		objdata, _, err := unstructured.UnstructuredJSONScheme.Decode(bytes, nil, nil)
		if err != nil {
			return nil, err
		}

		if l, ok := objdata.(*unstructured.UnstructuredList); ok {
			for _, item := range l.Items {
				copy := item
				result = append(result, &copy)
			}
		}
	}

	return result, nil
}

func toObjects(bytes []byte) ([]runtime.Object, error) {
	bytes, err := yaml.ToJSON(bytes)
	if err != nil {
		return nil, err
	}

	check := map[string]interface{}{}
	if err := json.Unmarshal(bytes, &check); err != nil || len(check) == 0 {
		return nil, err
	}

	obj, _, err := unstructured.UnstructuredJSONScheme.Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	if l, ok := obj.(*unstructured.UnstructuredList); ok {
		var result []runtime.Object
		for _, obj := range l.Items {
			copy := obj
			result = append(result, &copy)
		}
		return result, nil
	}

	return []runtime.Object{obj}, nil
}

type deployed struct {
	helmName      string
	helmNamespace string
	version       string
}

func Deployed(helmName, helmNamespace string) *deployed {
	return &deployed{
		helmName:      helmName,
		helmNamespace: helmNamespace,
	}
}

func (d *deployed) AllResources() ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured
	// manifest

	return resources, nil
}
