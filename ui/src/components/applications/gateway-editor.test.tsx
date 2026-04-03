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
  }: {
    checked?: boolean
    disabled?: boolean
    onCheckedChange?: (checked: boolean) => void
  }) => (
    <input
      aria-label="Enable public access"
      type="checkbox"
      checked={checked}
      disabled={disabled}
      onChange={(event) => onCheckedChange?.(event.target.checked)}
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
  ComboboxInput: (props: React.ComponentProps<"input">) => <input {...props} />,
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
    port: 80,
    protocol: "http",
    domain: "demo.example.com",
    path: "/",
    exposed: true,
    service_type: "ClusterIP",
    ...overrides,
  }
}

async function renderEditor(gateway?: GatewaySpec) {
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

async function clickElement(element: Element | null) {
  await act(async () => {
    element?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
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

      return { data: { items: [] } }
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("forces public access off and disables it when protocol changes to tcp", async () => {
    const { container, root } = await renderEditor(buildGateway())

    await clickElement(container.querySelector('[data-combobox-item="tcp"]'))

    const checkbox = container.querySelector('input[type="checkbox"]') as HTMLInputElement | null
    expect(checkbox).not.toBeNull()
    expect(checkbox?.disabled).toBe(true)
    expect(checkbox?.checked).toBe(false)

    await act(async () => {
      root.unmount()
    })
  })

  it("forces public access off and disables it when protocol changes to udp", async () => {
    const { container, root } = await renderEditor(buildGateway({ protocol: "https" }))

    await clickElement(container.querySelector('[data-combobox-item="udp"]'))

    const checkbox = container.querySelector('input[type="checkbox"]') as HTMLInputElement | null
    expect(checkbox).not.toBeNull()
    expect(checkbox?.disabled).toBe(true)
    expect(checkbox?.checked).toBe(false)

    await act(async () => {
      root.unmount()
    })
  })

  it("shows the unsupported public access tooltip copy for non-http protocols", async () => {
    const { container, root } = await renderEditor(buildGateway({ protocol: "tcp", exposed: false, domain: "", path: "" }))

    expect(container.textContent).toContain(
      "Public access is currently available only for HTTP/HTTPS gateways. TCP/UDP public exposure is not supported yet.",
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps public access enabled for http when Gateway API is installed", async () => {
    const { container, root } = await renderEditor(buildGateway({ protocol: "http", exposed: true }))

    const checkbox = container.querySelector('input[type="checkbox"]') as HTMLInputElement | null
    expect(checkbox).not.toBeNull()
    expect(checkbox?.disabled).toBe(false)
    expect(checkbox?.checked).toBe(true)
    expect(container.textContent).not.toContain(
      "Public access is currently available only for HTTP/HTTPS gateways. TCP/UDP public exposure is not supported yet.",
    )

    await act(async () => {
      root.unmount()
    })
  })
})
