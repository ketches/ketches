import { act, createContext, useContext } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockInvalidateQueries,
  mockMutate,
  mockUseQuery,
} = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockMutate: vi.fn(),
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/api/apps", () => ({
  appsApi: {
    addGateway: vi.fn(),
    updateGateway: vi.fn(),
  },
}))

vi.mock("@/api/certificates", () => ({
  certificatesApi: {
    listByCluster: vi.fn(),
    listByEnv: vi.fn(),
  },
}))

vi.mock("@/api/clusters", () => ({
  clustersApi: {
    getGatewayAPIStatus: vi.fn(),
  },
}))

vi.mock("@/api/domains", () => ({
  domainsApi: {
    listByCluster: vi.fn(),
    listByEnv: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: ({
    checked,
    disabled,
    onCheckedChange,
    ...props
  }: {
    checked?: boolean
    disabled?: boolean
    onCheckedChange?: (checked: boolean) => void
  } & React.ComponentProps<"input">) => (
    <input
      type="checkbox"
      checked={checked}
      disabled={disabled}
      onChange={(event) => onCheckedChange?.(event.target.checked)}
      {...props}
    />
  ),
}))

const ComboboxContext = createContext<{
  onValueChange?: (value: string | null) => void
}>({})

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({
    children,
    onValueChange,
  }: {
    children: React.ReactNode
    onValueChange?: (value: string | null) => void
  }) => (
    <ComboboxContext.Provider value={{ onValueChange }}>
      <div>{children}</div>
    </ComboboxContext.Provider>
  ),
  ComboboxInput: ({
    children,
    ...props
  }: React.ComponentProps<"input"> & { children?: React.ReactNode }) => (
    <div>
      <input {...props} />
      {children}
    </div>
  ),
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxLabel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxItem: ({
    children,
    value,
  }: {
    children: React.ReactNode
    value: string
  }) => {
    const { onValueChange } = useContext(ComboboxContext)
    return (
      <button type="button" data-combobox-item={value} onClick={() => onValueChange?.(value)}>
        {children}
      </button>
    )
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean, children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children, className }: { children: React.ReactNode; className?: string }) => <div className={className}>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldError: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode, htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/input-group", () => ({
  InputGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  InputGroupAddon: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  InputGroupInput: (props: React.ComponentProps<"input">) => <input {...props} />,
  InputGroupText: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/ui/item", () => ({
  Item: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: () => <hr />,
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

import type { App, GatewaySpec } from "@/api/apps"
import { GatewayEditor } from "./gateway-editor"

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
    env: {
      id: "env-1",
      name: "Dev",
      slug: "dev",
      description: "",
      project_id: "project-1",
      project_name: "Test Project",
      cluster_id: "cluster-1",
      cluster_name: "Test Cluster",
      cluster_namespace: "default",
      is_build_env: false,
      created_at: "2026-04-03T00:00:00Z",
    },
  }
}

function buildGateway(overrides: Partial<GatewaySpec> = {}): GatewaySpec {
  return {
    id: "gateway-1",
    app_id: "app-1",
    port: 80,
    protocol: "http",
    service_type: "ClusterIP",
    routes: [
      {
        id: "route-1",
        gateway_id: "gateway-1",
        host: "demo.example.com",
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

async function renderEditor(gateway?: GatewaySpec | null) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <GatewayEditor
        app={buildApp()}
        gateway={gateway}
        open
        onOpenChange={() => undefined}
      />,
    )
  })

  return { container, root }
}

function getMutatedGateway(): GatewaySpec {
  const firstCall = mockMutate.mock.calls[0]
  expect(firstCall).toBeDefined()
  return firstCall[0] as GatewaySpec
}

async function clickElement(element: Element | null | undefined) {
  await act(async () => {
    element?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

async function changeInput(input: HTMLInputElement, value: string) {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
  })
}

async function submitForm(container: HTMLElement) {
  const form = container.querySelector("form")
  expect(form).not.toBeNull()

  await act(async () => {
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  })
}

describe("GatewayEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks()
      ; (globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: string[] }) => {
      if (queryKey[0] === "cluster-gateway-api-status") {
        return { data: { installed: true } }
      }

      if (queryKey[0] === "env-domains") {
        return {
          data: {
            items: [
              {
                id: "env-domain-1",
                name: "Env Primary",
                domain: "*.env.example.com",
              },
            ],
          },
        }
      }

      if (queryKey[0] === "cluster-domains") {
        return {
          data: {
            items: [
              {
                id: "cluster-domain-1",
                name: "Cluster Primary",
                domain: "*.cluster.example.com",
              },
            ],
          },
        }
      }

      if (queryKey[0] === "cluster-certificates") {
        return {
          data: {
            items: [
              {
                id: "cluster-cert-1",
                name: "Cluster Wildcard",
                description: "Cluster certificate",
                scope: "cluster",
                cluster_id: "cluster-1",
                env_id: "",
                created_at: "2026-04-03T00:00:00Z",
              },
            ],
          },
        }
      }

      if (queryKey[0] === "env-certificates") {
        return {
          data: {
            items: [
              {
                id: "env-cert-1",
                name: "Env Wildcard",
                description: "Environment certificate",
                scope: "env",
                cluster_id: "cluster-1",
                env_id: "env-1",
                created_at: "2026-04-03T00:00:00Z",
              },
            ],
          },
        }
      }

      return { data: { items: [] } }
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("creates a gateway with multiple HTTP routes", async () => {
    const { container, root } = await renderEditor(null)

    const hostInputs = () => Array.from(container.querySelectorAll('input[aria-label^="Route host"]')) as HTMLInputElement[]
    await changeInput(hostInputs()[0], "api.example.com")
    await clickElement(Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Add Route")))
    await changeInput(hostInputs()[1], "admin.example.com")

    await submitForm(container)

    const payload = getMutatedGateway()
    expect(payload.routes).toHaveLength(2)
    expect(payload.routes?.map((route) => route.host)).toEqual(["api.example.com", "admin.example.com"])
    expect(payload.routes?.[0].backends?.[0]).toMatchObject({ backend_app_id: "app-1", backend_port: 80, weight: 100 })

    await act(async () => {
      root.unmount()
    })
  })

  it("requires a certificate for enabled HTTPS routes", async () => {
    const { container, root } = await renderEditor(buildGateway({
      routes: [
        {
          id: "route-1",
          host: "secure.example.com",
          listener_protocol: "https",
          path: "/",
          path_match_type: "PathPrefix",
          enabled: true,
          backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }],
        },
      ],
    }))

    await submitForm(container)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(container.textContent).toContain("TLS certificate is required for HTTPS")

    await act(async () => {
      root.unmount()
    })
  })

  it("preserves HTTP route drafts when switching protocol away and back while omitting routes from TCP submit", async () => {
    const { container, root } = await renderEditor(buildGateway())

    const hostInput = container.querySelector('input[aria-label="Route host 1"]') as HTMLInputElement | null
    expect(hostInput).not.toBeNull()
    await changeInput(hostInput!, "draft.example.com")

    await clickElement(container.querySelector('[data-combobox-item="tcp"]'))
    expect(container.querySelector('input[aria-label="Route host 1"]')).toBeNull()

    await submitForm(container)
    expect(getMutatedGateway().routes).toBeUndefined()

    mockMutate.mockClear()
    await clickElement(container.querySelector('[data-combobox-item="http"]'))
    const restoredInput = container.querySelector('input[aria-label="Route host 1"]') as HTMLInputElement | null
    expect(restoredInput?.value).toBe("draft.example.com")

    await act(async () => {
      root.unmount()
    })
  })

  it("serializes route timeout and backend weight settings", async () => {
    const { container, root } = await renderEditor(buildGateway())

    const requestTimeout = container.querySelector('input[aria-label="Route request timeout 1"]') as HTMLInputElement | null
    const backendTimeout = container.querySelector('input[aria-label="Route backend timeout 1"]') as HTMLInputElement | null
    const backendWeight = container.querySelector('input[aria-label="Route backend weight 1"]') as HTMLInputElement | null
    expect(requestTimeout).not.toBeNull()
    expect(backendTimeout).not.toBeNull()
    expect(backendWeight).not.toBeNull()

    await changeInput(requestTimeout!, "30s")
    await changeInput(backendTimeout!, "25s")
    await changeInput(backendWeight!, "75")
    await submitForm(container)

    const route = getMutatedGateway().routes?.[0]
    expect(route?.timeouts).toEqual({ request: "30s", backend_request: "25s" })
    expect(route?.backends?.[0]).toMatchObject({ backend_port: 80, weight: 75 })

    await act(async () => {
      root.unmount()
    })
  })

  it("deletes a route from the submitted payload", async () => {
    const { container, root } = await renderEditor(buildGateway({
      routes: [
        { id: "route-1", host: "api.example.com", listener_protocol: "http", path: "/", path_match_type: "PathPrefix", enabled: true, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
        { id: "route-2", host: "admin.example.com", listener_protocol: "http", path: "/admin", path_match_type: "PathPrefix", enabled: true, backends: [{ backend_app_id: "app-1", backend_port: 80, weight: 100 }] },
      ],
    }))

    await clickElement(container.querySelector('button[aria-label="Delete route 2"]'))
    await submitForm(container)

    const payload = getMutatedGateway()
    expect(payload.routes).toHaveLength(1)
    expect(payload.routes?.[0].host).toBe("api.example.com")

    await act(async () => {
      root.unmount()
    })
  })
})
