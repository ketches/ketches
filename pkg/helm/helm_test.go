package helm

import (
	"fmt"
	"testing"
)

func TestResourcesFromManifest(t *testing.T) {
	resources, err := ResourcesFromManifest("test", "default")
	if err != nil {
		fmt.Println(err)
	}
	for _, r := range resources {
		fmt.Println(r.Kind, r.Name)
	}
}
