package core

import (
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
