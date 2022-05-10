package conf

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

var opt *Option

func init() {
	opt = BuildOption()
}

type Option struct {
	KubeConfig string
}

func BuildOption() *Option {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.Parse()
	return &Option{
		KubeConfig: *kubeconfig,
	}
}

var (
	port     int
	basePath string
)

func Port() int {
	if port == 0 {
		port, _ = strconv.Atoi(os.Getenv("KETCHES_PORT"))
		if port == 0 {
			port = 80
		}
	}

	return port
}

func BasePath() string {
	basePath = os.Getenv("KETCHES_BASE_PATH")
	if len(basePath) == 0 {
		return basePath
	}

	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	return basePath
}
