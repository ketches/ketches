import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockGateways,
  mockProjectRole,
  mockUseQuery,
  mockWindowOpen,
} = vi.hoisted(() => ({
  mockGateways: [] as unknown[],
  mockProjectRole: vi.fn(),
  mockUseQuery: vi.fn(),
  mockWindowOpen: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("@/api/apps", () => ({
  appsApi: {
    listGateways: vi.fn(),
    deleteGateway: vi.fn(),
  },
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => mockProjectRole(),
}))

vi.mock("@/components/applications/gateway-editor", () => ({
  GatewayEditor: () => null,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title, description }: { title: string; description?: string }) => (
    <div>
      <span>{title}</span>
      <span>{description}</span>
    </div>
  ),
}))

vi.mock("../shared/color-badge", () => ({
  ColorBadge: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <span className={className}>{children}</span>
  ),
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({
    columns,
    data,
    leftToolbar,
    rightToolbar,
    sourceEmptyContent,
  }: {
    columns: Array<{
      id?: string
      accessorKey?: string
      cell?: unknown
    }>
    data: Array<Record<string, unknown>>
    leftToolbar?: (table: unknown) => React.ReactNode
    rightToolbar?: (table: unknown) => React.ReactNode
    sourceEmptyContent?: React.ReactNode
  }) => {
    if (data.length === 0) {
      return (
        <div>
          {leftToolbar?.({})}
          {rightToolbar?.({})}
          {sourceEmptyContent}
        </div>
      )
    }

    return (
      <div>
        {leftToolbar?.({})}
        {rightToolbar?.({})}
        <div data-testid="gateway-rows">
          {data.map((rowData, rowIndex) => (
            <div data-testid="gateway-row" key={`${rowData.id ?? rowIndex}`}>
              {columns
                .filter((column) => column.id !== "select" && column.id !== "actions")
                .map((column, columnIndex) => {
                  const row = {
                    original: rowData,
                    getIsSelected: () => false,
                    toggleSelected: () => undefined,
                  }

                  if (typeof column.cell === "function") {
                    return (
                      <div key={`${column.id ?? column.accessorKey ?? columnIndex}`}>
                        {column.cell({ row } as never)}
                      </div>
                    )
                  }

                  const value = column.accessorKey ? rowData[column.accessorKey] : ""
                  return <span key={`${column.id ?? column.accessorKey ?? columnIndex}`}>{String(value ?? "")}</span>
                })}
            </div>
          ))}
        </div>
      </div>
    )
  },
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogAction: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: (props: React.ComponentProps<"input">) => <input type="checkbox" {...props} />,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipTrigger: ({
    children,
    render,
  }: {
    children?: React.ReactNode
    render?: React.ReactElement
  }) => render ? render : <>{children}</>,
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import type { App, GatewaySpec } from "@/api/apps"
import { NetworkConfig } from "./gateway-table"

function buildApp(): App {
  return {
    id: "app-1",
    slug: "demo-app",
    name: "Demo App",
    description: "",
    env_id: "env-1",
    app_type: "Deployment",
    container_image: "ghcr.io/acme/demo:latest",
    replicas: 1,
    request_cpu: 100,
    request_memory: 128,
    limit_cpu: 500,
    limit_memory: 512,
    status: "running",
    available_actions: [],
    created_at: "2026-04-03T00:00:00Z",
  }
}

function buildGateway(overrides: Partial<GatewaySpec> = {}): GatewaySpec {
  return {
    id: "gateway-1",
    app_id: "app-1",
    port: 80,
    protocol: "http",
    service_type: "ClusterIP",
    internal_address: "demo.dev.svc:80",
    routes: [
      {
        id: "route-1",
        gateway_id: "gateway-1",
        host: "api.example.com",
        listener_protocol: "http",
        path: "/",
        path_match_type: "PathPrefix",
        enabled: true,
        backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }],
      },
    ],
    ...overrides,
  }
}

async function renderNetworkConfig(gateways: GatewaySpec[]) {
  mockGateways.splice(0, mockGateways.length, ...gateways)
  mockUseQuery.mockReturnValue({
    data: gateways,
    isLoading: false,
    refetch: vi.fn(),
  })

  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(<NetworkConfig app={buildApp()} />)
  })

  return { container, root }
}

async function changeInput(input: HTMLInputElement, value: string) {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
  })
}

async function clickButton(button: HTMLButtonElement | undefined) {
  await act(async () => {
    button?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

describe("NetworkConfig gateway table", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockProjectRole.mockReturnValue("owner")
      ; (globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    Object.defineProperty(window, "open", {
      value: mockWindowOpen,
      writable: true,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders one gateway row with multiple route badges and overflow count", async () => {
    const { container, root } = await renderNetworkConfig([
      buildGateway({
        routes: [
          { id: "route-1", host: "api.example.com", listener_protocol: "http", path: "/", path_match_type: "PathPrefix", enabled: true, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
          { id: "route-2", host: "admin.example.com", listener_protocol: "https", path: "/admin", path_match_type: "PathPrefix", enabled: true, cert_id: "cert-1", backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
          { id: "route-3", host: "beta.example.com", listener_protocol: "http", path: "/beta", path_match_type: "PathPrefix", enabled: false, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
          { id: "route-4", host: "extra.example.com", listener_protocol: "https", path: "/", path_match_type: "PathPrefix", enabled: true, cert_id: "cert-2", backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
        ],
      }),
    ])

    expect(container.querySelectorAll('[data-testid="gateway-row"]')).toHaveLength(1)
    expect(container.textContent).toContain("api.example.com")
    expect(container.textContent).toContain("admin.example.com/admin")
    expect(container.textContent).toContain("beta.example.com/beta")
    expect(container.textContent).toContain("+1")

    await act(async () => {
      root.unmount()
    })
  })

  it("filters gateways by route host and path", async () => {
    const { container, root } = await renderNetworkConfig([
      buildGateway({ id: "gateway-1", routes: [{ id: "route-1", host: "api.example.com", listener_protocol: "http", path: "/", path_match_type: "PathPrefix", enabled: true, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] }] }),
      buildGateway({ id: "gateway-2", port: 8080, routes: [{ id: "route-2", host: "reports.example.com", listener_protocol: "https", path: "/daily", path_match_type: "PathPrefix", enabled: true, cert_id: "cert-1", backends: [{ backend_app_id: "app-1", backend_port: 8080, weight: 100 }] }] }),
    ])

    const input = container.querySelector('input[placeholder="Filter by port, protocol, service, route..."]') as HTMLInputElement | null
    expect(input).not.toBeNull()

    await changeInput(input!, "reports")
    expect(container.querySelectorAll('[data-testid="gateway-row"]')).toHaveLength(1)
    expect(container.textContent).toContain("reports.example.com")
    expect(container.textContent).not.toContain("api.example.com")

    await changeInput(input!, "/daily")
    expect(container.querySelectorAll('[data-testid="gateway-row"]')).toHaveLength(1)
    expect(container.textContent).toContain("reports.example.com")

    await act(async () => {
      root.unmount()
    })
  })

  it("opens enabled route links with listener protocol and leaves disabled routes non-clickable", async () => {
    const { container, root } = await renderNetworkConfig([
      buildGateway({
        routes: [
          { id: "route-1", host: "secure.example.com", listener_protocol: "https", path: "/login", path_match_type: "PathPrefix", enabled: true, cert_id: "cert-1", backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
          { id: "route-2", host: "disabled.example.com", listener_protocol: "http", path: "/off", path_match_type: "PathPrefix", enabled: false, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
        ],
      }),
    ])

    const buttons = Array.from(container.querySelectorAll("button"))
    const enabledRouteButton = buttons.find((button) => button.textContent?.includes("secure.example.com/login"))
    const disabledRouteButton = buttons.find((button) => button.textContent?.includes("disabled.example.com/off"))

    expect(enabledRouteButton).toBeDefined()
    expect(disabledRouteButton).toBeUndefined()

    await clickButton(enabledRouteButton)
    expect(mockWindowOpen).toHaveBeenCalledWith("https://secure.example.com/login", "_blank")

    await act(async () => {
      root.unmount()
    })
  })

  it("hides write controls while the project role is unknown", async () => {
    mockProjectRole.mockReturnValue(null)

    const { container, root } = await renderNetworkConfig([buildGateway()])

    const addButton = Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent?.includes("Add Gateway"))
    expect(addButton).toBeUndefined()

    await act(async () => {
      root.unmount()
    })
  })
})
