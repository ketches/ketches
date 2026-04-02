/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { describe, expect, test, vi } from "vitest"
import { EditClusterConnectionDialog } from "./edit-cluster-connection-dialog"

const queryClient = new QueryClient()

describe("EditClusterConnectionDialog", () => {
  test("shows gateway host and update kubeconfig editing controls", async () => {
    const cluster = {
      id: "cluster-1",
      name: "My Cluster",
      gateway_host: "10.0.0.1",
      has_kube_config: true,
    }

    render(
      <QueryClientProvider client={queryClient}>
        <EditClusterConnectionDialog open={true} onOpenChange={vi.fn()} cluster={cluster as any} />
      </QueryClientProvider>
    )

    expect(await screen.findByRole("heading", { name: /Update KubeConfig/i })).toBeInTheDocument()

    const gatewayInput = await screen.findByPlaceholderText("e.g. 10.0.0.1 or gateway.example.com")
    expect(gatewayInput).toBeInTheDocument()
    expect(gatewayInput).toHaveValue("10.0.0.1")

    const kubeConfigTextarea = await screen.findByPlaceholderText(/Leave blank to keep existing configuration.../i)
    expect(kubeConfigTextarea).toBeInTheDocument()
    expect(kubeConfigTextarea).toHaveValue("")
  })
})
