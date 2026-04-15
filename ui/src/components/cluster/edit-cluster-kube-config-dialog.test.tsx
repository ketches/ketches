/** @vitest-environment jsdom */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import "@testing-library/jest-dom/vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, test, vi } from "vitest"
import { EditClusterKubeConfigDialog } from "./edit-cluster-kube-config-dialog"

const queryClient = new QueryClient()

afterEach(() => {
  cleanup()
})

describe("EditClusterConnectionDialog", () => {
  test("shows gateway host detection controls and edit kubeconfig heading", async () => {
    const cluster = {
      id: "cluster-1",
      name: "My Cluster",
      gateway_host: "10.0.0.1",
      has_kube_config: true,
    }

    render(
      <QueryClientProvider client={queryClient}>
        <EditClusterKubeConfigDialog open={true} onOpenChange={vi.fn()} cluster={cluster as any} />
      </QueryClientProvider>
    )

    expect(await screen.findByRole("heading", { name: /Edit KubeConfig/i })).toBeInTheDocument()

    const gatewayInput = await screen.findByPlaceholderText("e.g. 10.0.0.1 or gateway.example.com")
    expect(gatewayInput).toBeInTheDocument()
    expect(gatewayInput).toHaveValue("10.0.0.1")
    expect(screen.getByRole("button", { name: /detect/i })).toBeInTheDocument()

    const [kubeConfigTextarea] = await screen.findAllByPlaceholderText(/Leave blank to keep existing configuration.../i)
    expect(kubeConfigTextarea).toBeInTheDocument()
    expect(kubeConfigTextarea).toHaveValue("")
  })

  test("detects gateway host only when detect button is clicked", async () => {
    const cluster = {
      id: "cluster-1",
      name: "My Cluster",
      gateway_host: "10.0.0.1",
      has_kube_config: true,
    }

    render(
      <QueryClientProvider client={queryClient}>
        <EditClusterKubeConfigDialog open={true} onOpenChange={vi.fn()} cluster={cluster as any} />
      </QueryClientProvider>
    )

    const [kubeConfigTextarea] = await screen.findAllByPlaceholderText(/Leave blank to keep existing configuration.../i)
    const gatewayInput = await screen.findByPlaceholderText("e.g. 10.0.0.1 or gateway.example.com")

    fireEvent.change(kubeConfigTextarea, {
      target: {
        value: "clusters:\n  - cluster:\n      server: https://gateway.example.com:6443",
      },
    })

    expect(gatewayInput).toHaveValue("10.0.0.1")

    fireEvent.click(screen.getByRole("button", { name: /detect/i }))

    expect(gatewayInput).toHaveValue("gateway.example.com")
  })
})
