/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { describe, test, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { EditCodeRepositoryDialog } from "./edit-code-repository-dialog"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"

const queryClient = new QueryClient()

describe("EditCodeRepositoryDialog", () => {
  test("shows clear password action when repository has git password", async () => {
    const user = userEvent.setup()
    const repository = {
      id: "repo-1",
      name: "My Repo",
      git_url: "https://github.com/my/repo.git",
      git_username: "user",
      has_git_password: true, // The new API contract
    }

    render(
      <QueryClientProvider client={queryClient}>
        <EditCodeRepositoryDialog open={true} onOpenChange={() => {}} repo={repository as any} />
      </QueryClientProvider>
    )

    // Should not show the plaintext password field/echo
    expect(screen.queryByDisplayValue("********")).not.toBeInTheDocument()

    // Should show clear password action
    const clearBtn = await screen.findByRole("button", { name: /clear password/i })
    expect(clearBtn).toBeInTheDocument()

    // Assuming click it replaces state
    await user.click(clearBtn)
    
    // Expect input to become editable again
    expect(screen.getByLabelText(/password/i)).toBeEnabled()
  })
})
