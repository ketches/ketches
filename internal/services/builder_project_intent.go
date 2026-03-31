package services

import "strings"

type builderProjectIntent struct {
	ProjectKind                string
	Summary                    string
	SuggestedExecutorPolicyKey string
	SuggestedImageProfileKey   string
}

func AnalyzeBuilderProjectIntent(prompt string) builderProjectIntent {
	normalized := strings.ToLower(strings.TrimSpace(prompt))

	switch {
	case containsAnyBuilderIntentKeyword(normalized, "next.js", "nextjs", "ssr", "server-side render", "server side render"):
		return builderProjectIntent{
			ProjectKind:                "node_ssr_app",
			Summary:                    "Detected a Node SSR/web app request.",
			SuggestedExecutorPolicyKey: "workspace-node-ssr",
			SuggestedImageProfileKey:   "node-ssr",
		}
	case containsAnyBuilderIntentKeyword(normalized, "golang", "go api", "gin ", "fiber", "echo framework"):
		return builderProjectIntent{
			ProjectKind:                "go_api_service",
			Summary:                    "Detected a Go API/service request.",
			SuggestedExecutorPolicyKey: "workspace-go-api",
			SuggestedImageProfileKey:   "go-api",
		}
	case containsAnyBuilderIntentKeyword(normalized, "python", "fastapi", "django", "flask"):
		return builderProjectIntent{
			ProjectKind:                "python_api_service",
			Summary:                    "Detected a Python API/service request.",
			SuggestedExecutorPolicyKey: "workspace-python-api",
			SuggestedImageProfileKey:   "python-api",
		}
	case containsAnyBuilderIntentKeyword(normalized, "full-stack", "full stack", "frontend and backend", "backend and frontend"):
		return builderProjectIntent{
			ProjectKind:                "full_stack_app",
			Summary:                    "Detected a full-stack application request.",
			SuggestedExecutorPolicyKey: "workspace-full-stack",
			SuggestedImageProfileKey:   "full-stack",
		}
	default:
		return builderProjectIntent{
			ProjectKind:                "static_frontend_app",
			Summary:                    "Detected a static frontend application request.",
			SuggestedExecutorPolicyKey: "workspace-node-static",
			SuggestedImageProfileKey:   "node-static",
		}
	}
}

func containsAnyBuilderIntentKeyword(content string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}
