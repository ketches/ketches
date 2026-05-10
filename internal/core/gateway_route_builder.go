package core

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayHTTPRouteBuildInput struct {
	AppSlug   string
	AppID     string
	EnvID     string
	Namespace string
	GatewayID string
	RouteID   string
	Route     models.GatewayRouteSpec
	Backends  []GatewayHTTPBackendBuildInput
}

type GatewayHTTPBackendBuildInput struct {
	ServiceName string
	Port        int
	Weight      int32
}

func BuildGatewayHTTPRoute(input GatewayHTTPRouteBuildInput) *gatewayv1.HTTPRoute {
	if !input.Route.Enabled {
		return nil
	}

	sectionName := gatewayv1.SectionName("http")
	if strings.EqualFold(input.Route.ListenerProtocol, "https") {
		sectionName = buildSharedGatewayHTTPSListenerName(input.Route.Host)
	}

	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildGatewayHTTPRouteName(input.AppSlug, input.RouteID),
			Namespace: input.Namespace,
			Labels: map[string]string{
				kube.LabelManagedBy:      "true",
				kube.LabelAppID:          input.AppID,
				kube.LabelAppSlug:        input.AppSlug,
				kube.LabelEnvID:          input.EnvID,
				kube.LabelGatewayID:      input.GatewayID,
				kube.LabelGatewayRouteID: input.RouteID,
			},
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(SharedGatewayName()),
						Namespace:   ptrGatewayNamespace(gatewayv1.Namespace(SharedGatewayNamespace())),
						SectionName: ptrSectionName(sectionName),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{gatewayv1.Hostname(input.Route.Host)},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches:     buildGatewayHTTPRouteMatches(input.Route),
					Filters:     buildGatewayHTTPRouteFilters(input.Route.Filters),
					Timeouts:    buildGatewayHTTPRouteTimeouts(input.Route.Timeouts),
					BackendRefs: buildGatewayHTTPBackendRefs(input.Backends),
				},
			},
		},
	}
}

func buildGatewayHTTPRouteMatches(route models.GatewayRouteSpec) []gatewayv1.HTTPRouteMatch {
	pathMatchType := gatewayv1.PathMatchPathPrefix
	if route.PathMatchType == "Exact" {
		pathMatchType = gatewayv1.PathMatchExact
	}
	path := route.Path
	if path == "" {
		path = "/"
	}

	match := gatewayv1.HTTPRouteMatch{
		Path: &gatewayv1.HTTPPathMatch{
			Type:  &pathMatchType,
			Value: &path,
		},
	}
	if route.Matches != nil && route.Matches.Method != "" {
		method := gatewayv1.HTTPMethod(route.Matches.Method)
		match.Method = &method
	}
	return []gatewayv1.HTTPRouteMatch{match}
}

func buildGatewayHTTPRouteFilters(filters *models.GatewayRouteFilters) []gatewayv1.HTTPRouteFilter {
	if filters == nil {
		return nil
	}

	result := make([]gatewayv1.HTTPRouteFilter, 0, 2)
	if headerFilter := buildGatewayHTTPHeaderFilter(filters.RequestHeaders); headerFilter != nil {
		result = append(result, gatewayv1.HTTPRouteFilter{
			Type:                  gatewayv1.HTTPRouteFilterRequestHeaderModifier,
			RequestHeaderModifier: headerFilter,
		})
	}
	if headerFilter := buildGatewayHTTPHeaderFilter(filters.ResponseHeaders); headerFilter != nil {
		result = append(result, gatewayv1.HTTPRouteFilter{
			Type:                   gatewayv1.HTTPRouteFilterResponseHeaderModifier,
			ResponseHeaderModifier: headerFilter,
		})
	}
	return result
}

func buildGatewayHTTPHeaderFilter(modifier *models.GatewayHeaderModifier) *gatewayv1.HTTPHeaderFilter {
	if modifier == nil {
		return nil
	}

	filter := &gatewayv1.HTTPHeaderFilter{
		Remove: modifier.Remove,
	}
	for _, item := range modifier.Set {
		filter.Set = append(filter.Set, gatewayv1.HTTPHeader{
			Name:  gatewayv1.HTTPHeaderName(item.Name),
			Value: item.Value,
		})
	}
	for _, item := range modifier.Add {
		filter.Add = append(filter.Add, gatewayv1.HTTPHeader{
			Name:  gatewayv1.HTTPHeaderName(item.Name),
			Value: item.Value,
		})
	}
	if len(filter.Set) == 0 && len(filter.Add) == 0 && len(filter.Remove) == 0 {
		return nil
	}
	return filter
}

func buildGatewayHTTPRouteTimeouts(timeouts *models.GatewayRouteTimeouts) *gatewayv1.HTTPRouteTimeouts {
	if timeouts == nil {
		return nil
	}

	result := &gatewayv1.HTTPRouteTimeouts{}
	if strings.TrimSpace(timeouts.Request) != "" {
		duration := gatewayv1.Duration(strings.TrimSpace(timeouts.Request))
		result.Request = &duration
	}
	if strings.TrimSpace(timeouts.BackendRequest) != "" {
		duration := gatewayv1.Duration(strings.TrimSpace(timeouts.BackendRequest))
		result.BackendRequest = &duration
	}
	if result.Request == nil && result.BackendRequest == nil {
		return nil
	}
	return result
}

func buildGatewayHTTPBackendRefs(backends []GatewayHTTPBackendBuildInput) []gatewayv1.HTTPBackendRef {
	result := make([]gatewayv1.HTTPBackendRef, 0, len(backends))
	for _, backend := range backends {
		weight := backend.Weight
		result = append(result, gatewayv1.HTTPBackendRef{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(backend.ServiceName),
					Port: ptrPort(gatewayv1.PortNumber(backend.Port)),
				},
				Weight: &weight,
			},
		})
	}
	return result
}

func buildGatewayHTTPRouteName(appSlug, routeID string) string {
	name := sanitizeDNSLabel(fmt.Sprintf("%s-%s", appSlug, routeID))
	if len(name) <= 63 {
		return name
	}

	hash := sha1.Sum([]byte(routeID))
	suffix := hex.EncodeToString(hash[:])[:10]
	prefix := strings.TrimSuffix(name[:min(len(name), 52)], "-")
	return sanitizeDNSLabel(fmt.Sprintf("%s-%s", prefix, suffix))
}

var dnsLabelUnsafeChars = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeDNSLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = dnsLabelUnsafeChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "route"
	}
	return value
}
