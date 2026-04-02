/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { describe, test, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { EditPluginDialog } from "./edit-plugin-dialog"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

const queryClient = new QueryClient()

describe("EditPluginDialog", () => {
	test("shows clear field action for secrets", async () => {
		const pluginData = {
			id: "plugin-1",
			name: "My Plugin",
			registry_username: "robot",
			has_registry_password: true,
		}

		render(
      <QueryClientProvider client={queryClient}>
				<EditPluginDialog open={true} onOpenChange={() => {}} plugin={pluginData as any} projectId="project-1" />
      </QueryClientProvider>
		)

		const clearBtn = screen.getByText(/clear password/i)
		expect(clearBtn).toBeInTheDocument()
	})
})
