/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, test, vi } from "vitest"

import { ClusterIntegrationsConfig } from "./cluster-integrations-config"

const listIntegrationsMock = vi.fn()
const updateIntegrationMock = vi.fn()
const listNamespacesMock = vi.fn()
const listServicesWithPortsMock = vi.fn()

vi.mock("@/api/clusters", () => ({
  clustersApi: {
    listIntegrations: (...args: unknown[]) => listIntegrationsMock(...args),
    updateIntegration: (...args: unknown[]) => updateIntegrationMock(...args),
    listNamespaces: (...args: unknown[]) => listNamespacesMock(...args),
    listServicesWithPorts: (...args: unknown[]) => listServicesWithPortsMock(...args),
    createIntegration: vi.fn(),
    deleteIntegration: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

function renderConfig() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <ClusterIntegrationsConfig clusterId="cluster-1" />
    </QueryClientProvider>
  )
}

describe("ClusterIntegrationsConfig", () => {
  beforeEach(() => {
    listNamespacesMock.mockResolvedValue(["default"])
    listServicesWithPortsMock.mockResolvedValue([])
    updateIntegrationMock.mockResolvedValue({})
    listIntegrationsMock.mockResolvedValue([
      {
        id: "integration-1",
        cluster_id: "cluster-1",
        integration_type: "prometheus",
        name: "Prometheus",
        endpoint: "https://prom.example.com",
        username: "admin",
        has_password: true,
        has_token: true,
        has_ca_cert: true,
        skip_tls_verify: false,
        enabled: true,
        created_at: "2026-04-01T00:00:00Z",
      },
    ])
  })

  test("supports clearing stored integration secrets in edit mode", async () => {
    renderConfig()

    const editButton = await screen.findByRole("button", { name: /edit integration prometheus/i })
    fireEvent.click(editButton)

    expect(await screen.findByRole("heading", { name: /edit integration/i })).toBeInTheDocument()
    expect(screen.getAllByText("********").length).toBeGreaterThanOrEqual(2)

    fireEvent.click(screen.getByRole("button", { name: /clear password/i }))
    fireEvent.click(screen.getByRole("button", { name: /clear token/i }))
    fireEvent.click(screen.getByRole("button", { name: /clear ca certificate/i }))

    fireEvent.click(screen.getByRole("button", { name: /save changes/i }))

    await waitFor(() => {
      expect(updateIntegrationMock).toHaveBeenCalledWith(
        "cluster-1",
        "integration-1",
        expect.objectContaining({
          clear_password: true,
          clear_token: true,
          clear_ca_cert: true,
        })
      )
    })
  })
})
