import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { envsQueryState } = vi.hoisted(() => ({
  envsQueryState: {
    isFetched: false,
    isLoading: true,
    data: undefined as unknown,
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "envs") {
      return {
        data: envsQueryState.data,
        isLoading: envsQueryState.isLoading,
        isFetched: envsQueryState.isFetched,
        refetch: vi.fn(),
      }
    }

    return {
      data: undefined,
      isLoading: false,
      isFetched: true,
      refetch: vi.fn(),
    }
  },
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    hasHydrated: true,
    activeProjectId: "project-1",
    activeProjectName: "Demo Project",
    activeEnvId: null,
    setActiveContextWithNames: vi.fn(),
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { role: string } }) => unknown) =>
    selector({ user: { role: "user" } }),
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => "owner",
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => <div>Page Header</div>,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyEnvironmentState: () => <div>No environments yet</div>,
}))

vi.mock("@/components/applications/application-list", () => ({
  ApplicationList: () => <div>Application List</div>,
}))

vi.mock("@/components/applications/app-groups-view", () => ({
  AppGroupsView: () => <div>App Groups</div>,
}))

vi.mock("@/components/applications/create-app-dialog", () => ({
  CreateAppDialog: () => null,
}))

vi.mock("@/components/applications/create-app-group-dialog", () => ({
  CreateAppGroupDialog: () => null,
}))

vi.mock("@/components/applications/import-apps-dialog", () => ({
  ImportAppsDialog: () => null,
}))

vi.mock("@/components/environment/create-environment-dialog", () => ({
  CreateEnvironmentDialog: () => null,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: (props: React.ComponentProps<"input">) => <input {...props} />,
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
}))

import { ApplicationsPage } from "./applications-page"

describe("ApplicationsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    envsQueryState.isFetched = false
    envsQueryState.isLoading = true
    envsQueryState.data = undefined
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: {
        getItem: () => null,
        setItem: vi.fn(),
      },
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows a skeleton loading state while environments are loading", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ApplicationsPage />)
    })

    expect(container.querySelector('[data-testid="applications-loading"]')).not.toBeNull()
    expect(container.querySelector('[data-slot="skeleton"]')).not.toBeNull()
    expect(container.textContent).not.toContain("No environments yet")
    expect(container.textContent).not.toContain("Application List")

    await act(async () => {
      root.unmount()
    })
  })
})
