import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { waitFor } from "@testing-library/react"
import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockGetCapabilities } = vi.hoisted(() => ({
  mockGetCapabilities: vi.fn(),
}))

vi.mock("@/api/projects", () => ({
  projectsApi: {
    getCapabilities: (...args: unknown[]) => mockGetCapabilities(...args),
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: {
    user: { id: string; username: string; email: string; role: string }
  }) => unknown) => selector({
    user: {
      id: "user-1",
      username: "alice",
      email: "alice@example.com",
      role: "user",
    },
  }),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: (selector: (state: { activeProjectId: string | null }) => unknown) => selector({
    activeProjectId: "project-1",
  }),
}))

import { useProjectRole } from "./useProjectRole"

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function ProjectRoleReader({ projectId }: { projectId: string | null }) {
  const role = useProjectRole(projectId)
  return <span data-testid="project-role">{role ?? "unknown"}</span>
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
}

describe("useProjectRole", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("loads the current user's role from the project capabilities endpoint", async () => {
    mockGetCapabilities.mockResolvedValue({
      project_role: "developer",
      capabilities: { read: true, write: true, manage: false },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <QueryClientProvider client={createQueryClient()}>
          <ProjectRoleReader projectId="project-1" />
        </QueryClientProvider>
      )
      await flushPromises()
    })

    expect(mockGetCapabilities).toHaveBeenCalledWith("project-1")
    await waitFor(() => expect(container.textContent).toBe("developer"))

    await act(async () => root.unmount())
  })

  it("does not let a late response overwrite the role for a newly selected project", async () => {
    let resolveFirstProject: ((value: {
      project_role: string
      capabilities: { read: boolean; write: boolean; manage: boolean }
    }) => void) | undefined
    mockGetCapabilities.mockImplementation((projectId: string) => {
      if (projectId === "project-1") {
        return new Promise((resolve) => {
          resolveFirstProject = resolve
        })
      }
      return Promise.resolve({
        project_role: "viewer",
        capabilities: { read: true, write: false, manage: false },
      })
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const queryClient = createQueryClient()

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <ProjectRoleReader projectId="project-1" />
        </QueryClientProvider>
      )
    })
    expect(container.textContent).toBe("unknown")

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <ProjectRoleReader projectId="project-2" />
        </QueryClientProvider>
      )
      await flushPromises()
    })
    await waitFor(() => expect(container.textContent).toBe("viewer"))

    await act(async () => {
      resolveFirstProject?.({
        project_role: "owner",
        capabilities: { read: true, write: true, manage: true },
      })
      await flushPromises()
    })
    await waitFor(() => expect(container.textContent).toBe("viewer"))

    await act(async () => root.unmount())
  })

  it("returns an unknown role when the capabilities request fails", async () => {
    mockGetCapabilities.mockRejectedValue(new Error("request failed"))

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <QueryClientProvider client={createQueryClient()}>
          <ProjectRoleReader projectId="project-1" />
        </QueryClientProvider>
      )
      await flushPromises()
    })

    expect(container.textContent).toBe("unknown")

    await act(async () => root.unmount())
  })

  it("fails closed when a background capabilities refetch fails", async () => {
    mockGetCapabilities
      .mockResolvedValueOnce({
        project_role: "developer",
        capabilities: { read: true, write: true, manage: false },
      })
      .mockRejectedValueOnce(new Error("request failed"))

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const queryClient = createQueryClient()

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <ProjectRoleReader projectId="project-1" />
        </QueryClientProvider>
      )
    })
    await waitFor(() => expect(container.textContent).toBe("developer"))

    await act(async () => {
      await queryClient.invalidateQueries({
        queryKey: ["project-capabilities", "project-1", "user-1", "user"],
      })
    })
    await waitFor(() => expect(container.textContent).toBe("unknown"))

    await act(async () => root.unmount())
  })

  it("stays unknown when the requested project is not loaded yet", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <QueryClientProvider client={createQueryClient()}>
          <ProjectRoleReader projectId={null} />
        </QueryClientProvider>
      )
      await flushPromises()
    })

    expect(container.textContent).toBe("unknown")
    expect(mockGetCapabilities).not.toHaveBeenCalled()

    await act(async () => root.unmount())
  })
})
