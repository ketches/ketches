package core

import (
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
)

type GatewayProviderCapabilities struct {
	RequestBodySizeLimit bool
	KeepAlive            bool
	WebSocket            bool
	Retry                bool
	SessionPersistence   bool
}

type GatewayExtensionInput struct {
	AppID     string
	GatewayID string
	RouteID   string
	Route     models.GatewayRouteSpec
}

type GatewayProviderAdapter interface {
	Capabilities() GatewayProviderCapabilities
	ValidateExtension(input GatewayExtensionInput) error
}

type defaultGatewayProviderAdapter struct{}

func (defaultGatewayProviderAdapter) Capabilities() GatewayProviderCapabilities {
	return GatewayProviderCapabilities{}
}

func (defaultGatewayProviderAdapter) ValidateExtension(input GatewayExtensionInput) error {
	if input.Route.Extension == nil {
		return nil
	}
	if input.Route.Extension.RequestBodySize != "" {
		return unsupportedGatewayProviderFieldError("request_body_size")
	}
	if input.Route.Extension.KeepAlive != nil {
		return unsupportedGatewayProviderFieldError("keep_alive")
	}
	if input.Route.Extension.WebSocket {
		return unsupportedGatewayProviderFieldError("websocket")
	}
	return nil
}

func unsupportedGatewayProviderFieldError(field string) error {
	return app.NewErrorf("gateway provider does not support %s", field)
}
