/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { describe, test, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { ContainerRegistryDialog } from "./container-registry-dialog"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"

const queryClient = new QueryClient()

describe("ContainerRegistryDialog (Edit Mode)", () => {
  test("shows clear password action when registry has password", async () => {
    const user = userEvent.setup()
    const registry = {
      id: "reg-1",
      name: "My Registry",
      server: "registry.example.com",
      username: "user",
      has_password: true, // The new API contract
    }

    render(
      <QueryClientProvider client={queryClient}>
        <ContainerRegistryDialog open={true} onOpenChange={() => {}} scope="project" scopeId="1" registry={registry as any} />
      </QueryClientProvider>
    )

    expect(screen.queryByDisplayValue("********")).not.toBeInTheDocument()

    const clearBtn = await screen.findByRole("button", { name: /clear password/i })
    expect(clearBtn).toBeInTheDocument()

    await user.click(clearBtn)
    expect(screen.getByLabelText(/password/i)).toBeEnabled()
  })
})
