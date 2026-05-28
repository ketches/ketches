import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { Project } from "@/api/projects"

const { mockNavigate, mockSetActiveContextWithNames, testState } = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockSetActiveContextWithNames: vi.fn(),
  testState: {
    activeProjectId: "project-2",
    projects: [] as Project[],
  },
}))

vi.mock("@/api/projects", () => ({
  projectsApi: {
    delete: vi.fn(),
    listSimple: vi.fn(),
  },
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    hasHydrated: true,
    activeProjectId: testState.activeProjectId,
    setActiveContextWithNames: mockSetActiveContextWithNames,
  }),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    isPending: false,
    mutate: vi.fn(),
  }),
  useQuery: () => ({
    data: testState.projects,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("@/components/project/create-project-dialog", () => ({
  CreateProjectDialog: () => null,
}))

vi.mock("@/components/project/edit-project-dialog", () => ({
  EditProjectDialog: () => null,
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogAction: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  AlertDialogCancel: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div data-testid="dropdown-content">{children}</div>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <div data-testid="dropdown-group">{children}</div>,
  DropdownMenuItem: ({ children, variant, ...props }: React.ComponentProps<"div"> & { variant?: string }) => (
    <div role="menuitem" data-variant={variant} {...props}>
      {children}
    </div>
  ),
  DropdownMenuLabel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSeparator: () => <hr />,
  DropdownMenuTrigger: ({ children, render }: { children?: React.ReactNode; render?: React.ReactElement }) => render ?? <>{children}</>,
}))

vi.mock("@/components/ui/sidebar", () => ({
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuButton: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useSidebar: () => ({ isMobile: false }),
}))

vi.mock("lucide-react", () => {
  const icon =
    (testId: string) =>
    ({ className }: { className?: string }) => (
      <svg aria-hidden="true" className={className} data-testid={testId} />
    )

  return {
    ArrowRight: icon("arrow-right-icon"),
    Check: icon("check-icon"),
    ChevronsUpDown: icon("chevrons-up-down-icon"),
    GalleryVerticalEnd: icon("gallery-vertical-end-icon"),
    MoreVertical: icon("more-vertical-icon"),
    Pencil: icon("pencil-icon"),
    Plus: icon("plus-icon"),
    Trash2: icon("trash-icon"),
  }
})

import { ProjectSwitcher } from "./project-switcher"

function makeProject(id: string, name: string): Project {
  return {
    id,
    name,
    slug: id,
    collaboration_enabled: true,
    created_at: "2026-01-01T00:00:00Z",
  }
}

describe("ProjectSwitcher", () => {
  beforeEach(() => {
    testState.activeProjectId = "project-2"
    testState.projects = [
      makeProject("project-1", "Project One"),
      makeProject("project-2", "Project Two"),
      makeProject("project-3", "Project Three"),
    ]
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("moves the current project to the top and highlights its row text and icon in green", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ProjectSwitcher />)
    })

    const projectGroup = container.querySelector('[data-testid="dropdown-group"]')
    const projectRows = Array.from(projectGroup?.children ?? []).filter(
      (element) => element.getAttribute("role") === "menuitem"
    )

    expect(projectRows.map((row) => projectNameForRow(row))).toEqual([
      "Project Two",
      "Project One",
      "Project Three",
    ])
    expect(projectRows[0].getAttribute("class")).toContain("bg-green-500/10")
    expect(projectRows[0].getAttribute("class")).toContain("focus:**:!text-green-600")
    expect(projectRows[0].querySelector('[data-testid="check-icon"]')).toBeNull()
    expect(projectIconContainerForRow(projectRows[0])?.getAttribute("class")).toContain("!border-green-600")
    expect(projectIconContainerForRow(projectRows[0])?.getAttribute("class")).toContain("!text-green-600")
    expect(projectRows[0].querySelector('[data-testid="gallery-vertical-end-icon"]')?.getAttribute("class")).toContain("!text-green-600")
    expect(projectNameElementForRow(projectRows[0])?.getAttribute("class")).toContain("!text-green-600")

    await act(async () => {
      root.unmount()
    })
  })
})

function projectNameForRow(row: Element) {
  const text = row.textContent ?? ""
  const project = testState.projects.find((item) => text.includes(item.name))

  return project?.name
}

function projectNameElementForRow(row: Element) {
  return Array.from(row.querySelectorAll("span")).find(
    (element) => element.textContent === projectNameForRow(row)
  )
}

function projectIconContainerForRow(row: Element) {
  return row.querySelector('[data-testid="gallery-vertical-end-icon"]')?.parentElement
}
