package kube

// import (
// 	"encoding/json"
// 	"errors"
// 	"sync"

// 	"github.com/dgrijalva/jwt-go"
// 	dbtypes "github.com/ketches/ketches/db/types"
// 	"k8s.io/apimachinery/pkg/util/yaml"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"
// 	"k8s.io/client-go/tools/clientcmd"
// 	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
// 	apilatest "k8s.io/client-go/tools/clientcmd/api/latest"
// 	apiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
// 	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
// )

// const (
// 	ketchesSystemNamespace     = "ketches-system"
// 	ketchesAdminServiceAccount = "ketches-admin"
// )

// func init() {
// 	SetClusterMap()
// }

// var clusterMap = &sync.Map{}

// func SetClusterMap() {
// 	var clusters []dbtypes.Cluster
// 	err := repos.Cluster().All(&clusters)
// 	if err != nil {
// 		log.Errorf("get clusters failed. ERROR: %s", err.Error())
// 		return
// 	}

// 	for _, c := range clusters {
// 		kubeCluster, err := GetCluster(c)
// 		if err != nil {
// 			log.Error(err.Error())
// 			continue
// 		}
// 		clusterMap.Store(c.ID, kubeCluster)
// 	}
// }

// func GetCluster(c dbtypes.Cluster) (*models.KubeCluster, error) {
// 	var (
// 		kubeCluster *models.KubeCluster
// 		err         error
// 	)

// 	switch c.ImportMode {
// 	case enums.KubeClusterImportModeKubeConfig:
// 		switch c.Formatting {
// 		case enums.FormattingJson:
// 			kubeCluster, err = getClusterFromKubeConfigJson(c.KubeConfig)
// 		case enums.FormattingYaml:
// 			kubeCluster, err = getClusterFromKubeConfigYaml(c.KubeConfig)
// 		}
// 	case enums.KubeClusterImportModeServiceAccountToken:
// 		p := models.KubeServiceAccountTokenProfile{
// 			Name:        c.Name,
// 			Server:      c.Server,
// 			Certificate: c.Certificate,
// 			Token:       c.Token,
// 		}
// 		kubeCluster, err = getClusterFromServiceAccountToken(p)
// 	default:
// 		err = errors.New("not found kubernetes imported")
// 	}
// 	return kubeCluster, err
// }

// func getClusterFromServiceAccountToken(clusterConfig models.KubeServiceAccountTokenProfile) (*models.KubeCluster, error) {
// 	claims, err := parseSecretAccountToken(clusterConfig.Token)
// 	if err != nil {
// 		log.Errorf("parse services account token failed: %s", err.Error())
// 		return nil, err
// 	}
// 	authUser := claims.Subject
// 	c := apiv1.Config{
// 		APIVersion: "v1",
// 		Clusters: []apiv1.NamedCluster{
// 			{
// 				Name: clusterConfig.Name,
// 				Cluster: apiv1.Cluster{
// 					//CertificateAuthority: clusterConfig.Credential,
// 					CertificateAuthorityData: []byte(clusterConfig.Certificate),
// 					Server:                   clusterConfig.Server,
// 				},
// 			},
// 		},
// 		Contexts: []apiv1.NamedContext{
// 			{
// 				Context: apiv1.Context{
// 					Cluster:  clusterConfig.Name,
// 					AuthInfo: authUser,
// 				},
// 				Name: clusterConfig.Name,
// 			},
// 		},
// 		CurrentContext: clusterConfig.Name,
// 		Kind:           "Config",
// 		AuthInfos: []apiv1.NamedAuthInfo{
// 			{
// 				Name: authUser,
// 				AuthInfo: apiv1.AuthInfo{
// 					Token: clusterConfig.Token,
// 				},
// 			},
// 		},
// 	}

// 	return getClusterFromapiv1Config(&c)
// }

// func parseSecretAccountToken(token string) (*jwt.StandardClaims, error) {
// 	Claims := &jwt.StandardClaims{}
// 	_, err := jwt.ParseWithClaims(token, Claims, nil)
// 	return Claims, err
// }

// func getClusterFromKubeConfigJson(kubeConfigJsonString string) (*models.KubeCluster, error) {
// 	var c apiv1.Config
// 	err := json.Unmarshal([]byte(kubeConfigJsonString), &c)
// 	if err != nil {
// 		log.Errorf("json unmarshal failed.")
// 		return nil, err
// 	}

// 	return getClusterFromapiv1Config(&c)
// }

// func getClusterFromKubeConfigYaml(kubeConfigYamlString string) (*models.KubeCluster, error) {
// 	var c apiv1.Config
// 	err := yaml.Unmarshal([]byte(kubeConfigYamlString), &c)
// 	if err != nil {
// 		log.Errorf("yaml unmarshal failed.")
// 		return nil, err
// 	}

// 	return getClusterFromapiv1Config(&c)
// }

// func getClusterFromapiv1Config(c *apiv1.Config) (*models.KubeCluster, error) {
// 	configObject, err := apilatest.Scheme.ConvertToVersion(c, clientcmdapi.SchemeGroupVersion)
// 	configInternal := configObject.(*clientcmdapi.Config)

// 	clientConfig, err := clientcmd.NewDefaultClientConfig(*configInternal, &clientcmd.ConfigOverrides{
// 		ClusterDefaults: clientcmdapi.Cluster{
// 			Server: c.Clusters[0].Cluster.Server,
// 		},
// 	}).ClientConfig()

// 	if err != nil {
// 		log.Errorf("initial kubernetes client config error. ERROR: %s ", err.Error())
// 		return nil, err
// 	}

// 	clientConfig.QPS = KubeDefaultQPS
// 	clientConfig.Burst = KubeDefaultBurst

// 	clientSet, err := kubernetes.NewForConfig(clientConfig)
// 	if err != nil {
// 		log.Errorf("initial kubernetes client set error. ERROR: %s", err.Error())
// 		return nil, err
// 	}

// 	metricsClientSet, err := metrics.NewForConfig(clientConfig)
// 	if err != nil {
// 		log.Errorf("initial metrics client set error. ERROR: %s", err.Error())
// 	}

// 	return &models.KubeCluster{
// 		KubeConfig:        clientConfig,
// 		KubeClient:        clientSet,
// 		KubeMetricsClient: metricsClientSet,
// 	}, nil
// }

// func Cluster(clusterId string) *models.KubeCluster {
// 	kubeCluster, ok := clusterMap.Load(clusterId)
// 	if !ok {
// 		return nil
// 	}

// 	return kubeCluster.(*models.KubeCluster)
// }

// func Client(clusterId string) *kubernetes.Clientset {
// 	return Cluster(clusterId).KubeClient
// }

// func Config(clusterId string) *rest.Config {
// 	return Cluster(clusterId).KubeConfig
// }

// func MetricsClient(clusterId string) *metrics.Clientset {
// 	return Cluster(clusterId).KubeMetricsClient
// }

// func InitialCluster(kubeClient *kubernetes.Clientset) error {
// 	// create ketches-system namespace
// 	err := ApplyNamespace(kubeClient, models.KubeNamespace{
// 		Name: ketchesSystemNamespace,
// 	})
// 	if err != nil {
// 		log.Errorf("apply namespace [%s] failed.", ketchesSystemNamespace)
// 		return err
// 	}

// 	// create services account
// 	err = ApplyServiceAccount(kubeClient, models.KubeServiceAccount{
// 		Name:        ketchesAdminServiceAccount,
// 		Namespace:   ketchesSystemNamespace,
// 		BindingRole: true,
// 		ClusterRole: true,
// 		RoleName:    "cluster-admin",
// 	})
// 	if err != nil {
// 		log.Errorf("apply services account [%s] failed.", ketchesAdminServiceAccount)
// 		return err
// 	}

// 	// TODO: extensions

// 	return nil
// }
