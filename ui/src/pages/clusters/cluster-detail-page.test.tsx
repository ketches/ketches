/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { afterEach, describe, test, expect, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { ClusterDetailPage } from "./cluster-detail-page"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter, Route, Routes } from "react-router-dom"

const queryClient = new QueryClient()

afterEach(() => {
  cleanup()
})

vi.mock("@/api/clusters", () => ({
  clustersApi: {
    get: vi.fn().mockResolvedValue({
      id: "cluster-1",
      slug: "cluster-1",
      name: "My Cluster",
      enabled: true,
      connection_status: "connected",
      last_checked_at: "2026-04-02T00:00:00Z",
      api_server: "https://10.0.0.1:6443",
      has_kube_config: true,
      gateway_host: "gateway.example.com",
    }),
    listNodes: vi.fn().mockResolvedValue([]),
    listNamespaces: vi.fn().mockResolvedValue([]),
    listSimple: vi.fn().mockResolvedValue([]),
    checkConnectivity: vi.fn().mockResolvedValue({}),
    delete: vi.fn().mockResolvedValue({}),
  }
}))

vi.mock("@/contexts/bottom-panel-context", () => ({
	useBottomPanel: () => ({ openPanel: vi.fn(), closePanel: vi.fn(), isOpen: false })
}))

vi.mock("@/components/layout/page-header", () => ({
	PageHeader: () => <div>Page Header</div>,
}))

describe("ClusterDetailPage", () => {
  test("does not render a General tab and shows merged cluster information inside Overview", async () => {
    render(
      <QueryClientProvider client={queryClient}>
		<MemoryRouter initialEntries={["/clusters/cluster-1"]}>
			<Routes>
				<Route path="/clusters/:clusterId" element={<ClusterDetailPage />} />
			</Routes>
		</MemoryRouter>
      </QueryClientProvider>
    )

    expect(screen.queryByRole("tab", { name: /general/i })).not.toBeInTheDocument()

		expect(await screen.findByText(/cluster information/i)).toBeInTheDocument()
		expect(screen.getByText(/api server/i)).toBeInTheDocument()
		expect(screen.getByText("https://10.0.0.1:6443")).toBeInTheDocument()
		expect(screen.getByText(/^KubeConfig$/)).toBeInTheDocument()
		expect(screen.getByText(/configured/i)).toBeInTheDocument()
		expect(screen.getByText(/gateway host/i)).toBeInTheDocument()
		expect(screen.getByText("gateway.example.com")).toBeInTheDocument()
		expect(screen.getByRole("tab", { name: /domains/i })).toBeInTheDocument()
	})

  test("opens edit kubeconfig dialog from the kubeconfig action", async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/clusters/cluster-1"]}>
          <Routes>
            <Route path="/clusters/:clusterId" element={<ClusterDetailPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    )

    const buttons = await screen.findAllByRole("button", { name: /edit kubeconfig/i })
    fireEvent.click(buttons[buttons.length - 1]!)

    expect(await screen.findByRole("heading", { name: /edit kubeconfig/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /detect/i })).toBeInTheDocument()
  })
})
