/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { describe, test, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { EditClusterDialog } from "./edit-cluster-dialog"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"

const queryClient = new QueryClient()

describe("EditClusterDialog", () => {
  test("does not show kubeconfig and gateway editing controls", async () => {
    const user = userEvent.setup()
    const cluster = {
      id: "cluster-1",
      name: "My Cluster",
      gateway: "https://gateway.example.com",
    }

    render(
      <QueryClientProvider client={queryClient}>
        <EditClusterDialog open={true} onOpenChange={() => {}} cluster={cluster as any} />
      </QueryClientProvider>
    )

    // Try clicking Credentials tab if it exists
    const credsTab = screen.queryByRole("button", { name: /credentials/i })
    if (credsTab) await user.click(credsTab)

    // Kubeconfig and Gateway labels/inputs should no longer be present
    const hasGateway = screen.queryAllByText(/gateway/i).length > 0 || screen.queryByRole("textbox", { name: /gateway/i }) !== null
    expect(hasGateway).toBe(false)
    
    const hasKubeconfig = screen.queryAllByText(/kubeconfig/i).length > 0 || screen.queryByRole("textbox", { name: /kubeconfig/i }) !== null
    expect(hasKubeconfig).toBe(false)
  })
})
