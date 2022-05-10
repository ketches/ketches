package k8s

import (
	"github.com/ketches/ketches/conf"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	client *kubernetes.Clientset
	config *rest.Config
)

func GetClient() *kubernetes.Clientset {
	if client == nil {
		config = GetConfig()
		cli, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
		client = cli
	}
	return client
}

func GetConfig() *rest.Config {
	if config == nil {
		opt := conf.BuildOption()
		conf, err := clientcmd.BuildConfigFromFlags("", opt.KubeConfig)
		if err != nil {
			conf, err = rest.InClusterConfig()
			if err != nil {
				panic(err.Error())
			}
			config = conf
		}
	}
	return config
}
