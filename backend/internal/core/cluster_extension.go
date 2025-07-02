package core

import (
	"context"
	"log"
	"time"

	"github.com/ketches/ketches/internal/app"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NativeExtension struct {
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description,omitempty"`
	Installed   bool      `json:"installed"`
	Version     string    `json:"version,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

func CheckNativeExtensions(ctx context.Context, cli client.Client) []NativeExtension {
	var result []NativeExtension
	for _, ext := range nativeExtensions {
		if res, err := ext.Check(ctx, cli); err == nil {
			result = append(result, *res)
		}
	}
	return result
}

var nativeExtensions = map[string]nativeExtensionChecker{
	"gateway-api": &gatewayAPIExtension{},
}

type nativeExtensionChecker interface {
	Check(ctx context.Context, cli client.Client) (*NativeExtension, app.Error)
}

type gatewayAPIExtension struct{}

func (g *gatewayAPIExtension) Check(ctx context.Context, cli client.Client) (*NativeExtension, app.Error) {
	var gatewayCRD apiextensionsv1.CustomResourceDefinition
	if err := cli.Get(ctx, client.ObjectKey{Name: "gateways.gateway.networking.k8s.io"}, &gatewayCRD); err != nil {
		if k8serrors.IsNotFound(err) {
			return &NativeExtension{Slug: "gateway-api", DisplayName: "Gateway API", Description: "Enables the Gateway API for managing application gateways.", Installed: false}, nil
		}
		log.Println("failed to check Gateway API CRD:", err)
		return nil, app.ErrClusterOperationFailed
	}
	return &NativeExtension{
		Slug:        "gateway-api",
		DisplayName: "Gateway API",
		Description: "Enables the Gateway API for managing application gateways.",
		Installed:   true,
		Version:     gatewayCRD.Annotations["gateway.networking.k8s.io/bundle-version"],
		CreatedAt:   gatewayCRD.CreationTimestamp.Time,
	}, nil
}

func CheckGatewayAPIInstalled(ctx context.Context, cli client.Client) (bool, app.Error) {
	_, err := nativeExtensions["gateway-api"].Check(ctx, cli)
	if err != nil {
		return false, err
	}
	return true, nil
}
