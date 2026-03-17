import { act, createContext, useContext } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries, mockRefetch, mockUseQuery } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockRefetch: vi.fn<() => Promise<{ data: { repository: string, current_tag: string, tags: string[] } }>>(),
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: mockUseQuery,
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/api/apps", () => ({
  appsApi: {
    listImageTags: vi.fn(),
    updateImage: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({
    open,
    children,
  }: {
    open: boolean
    children: React.ReactNode
  }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <h1>{children}</h1>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({
    children,
    htmlFor,
    className,
  }: {
    children: React.ReactNode
    htmlFor?: string
    className?: string
  }) => <label htmlFor={htmlFor} className={className}>{children}</label>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({
    render,
    children,
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  }) => <>{render ?? children ?? null}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

const ComboboxContext = createContext<{
  onValueChange?: (value: string | null) => void
}>({})

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({
    children,
    onValueChange,
    open,
  }: {
    children: React.ReactNode
    onValueChange?: (value: string | null) => void
    open?: boolean
  }) => (
    <ComboboxContext.Provider value={{ onValueChange }}>
      <div data-combobox-open={open ? "true" : "false"}>{children}</div>
    </ComboboxContext.Provider>
  ),
  ComboboxTrigger: ({ children }: { children?: React.ReactNode }) => <button type="button">{children}</button>,
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
      <button
        type="button"
        data-combobox-item={value}
        onClick={() => onValueChange?.(value)}
      >
        {children}
      </button>
    )
  },
}))

vi.mock("@/components/ui/input-group", () => ({
  InputGroupButton: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

import type { App } from "@/api/apps"
import { ImageEditor } from "./image-editor"

function buildApp(overrides: Partial<App> = {}): App {
  return {
    id: "app-1",
    slug: "demo",
    name: "Demo App",
    description: "",
    env_id: "env-1",
    app_type: "Deployment",
    container_image: "ghcr.io/acme/demo:latest",
    registry_username: "",
    registry_password: "",
    replicas: 1,
    request_cpu: 100,
    request_memory: 128,
    limit_cpu: 500,
    limit_memory: 512,
    status: "running",
    available_actions: [],
    created_at: "2026-03-16T00:00:00Z",
    ...overrides,
  }
}

async function renderEditor(app: App) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <ImageEditor
        open
        onOpenChange={() => undefined}
        app={app}
      />
    )
  })

  return { container, root }
}

const clickElement = async (element: HTMLElement) => {
  await act(async () => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

describe("ImageEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    mockRefetch.mockResolvedValue({
      data: {
        repository: "ghcr.io/acme/demo",
        current_tag: "latest",
        tags: ["latest"],
      },
    })
    mockUseQuery.mockImplementation(() => ({
      data: {
        repository: "ghcr.io/acme/demo",
        current_tag: "latest",
        tags: ["latest"],
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: mockRefetch,
    }))
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows credentials immediately when the app already has registry credentials", async () => {
    const { container, root } = await renderEditor(buildApp({
      registry_username: "robot",
    }))

    expect(container.textContent).toContain("Registry Username")
    expect(container.textContent).toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps credentials collapsed when the app has none until the key button is clicked", async () => {
    const { container, root } = await renderEditor(buildApp())

    expect(container.textContent).not.toContain("Registry Username")
    expect(container.textContent).not.toContain("Registry Password")

    const toggle = container.querySelector('button[aria-label="Registry credentials"]') as HTMLButtonElement | null

    expect(toggle).not.toBeNull()

    await clickElement(toggle as HTMLButtonElement)

    expect(container.textContent).toContain("Registry Username")
    expect(container.textContent).toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps image loading manual and opens the combobox after refreshing options", async () => {
    const { container, root } = await renderEditor(buildApp())

    expect(container.textContent).not.toContain("Image Tag")
    expect(mockUseQuery).toHaveBeenCalledWith(expect.objectContaining({
      enabled: false,
    }))
    expect(mockRefetch).not.toHaveBeenCalled()
    expect(container.querySelector('[data-combobox-open="true"]')).toBeNull()

    const refreshButton = container.querySelector('button[aria-label="Refresh image options"]') as HTMLButtonElement | null
    const imageInput = container.querySelector('input[name="container_image"]') as HTMLInputElement | null

    expect(refreshButton).not.toBeNull()
    expect(imageInput).not.toBeNull()

    await clickElement(refreshButton as HTMLButtonElement)
    await act(async () => {
      await Promise.resolve()
    })

    expect(mockRefetch).toHaveBeenCalledTimes(1)
    expect(container.querySelector('[data-combobox-open="true"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("shows full image names in the combobox options and updates the input when selected", async () => {
    const { container, root } = await renderEditor(buildApp())

    expect(container.textContent).toContain("ghcr.io/acme/demo:latest")

    const option = container.querySelector('[data-combobox-item="ghcr.io/acme/demo:latest"]') as HTMLButtonElement | null

    expect(option).not.toBeNull()

    await clickElement(option as HTMLButtonElement)

    expect((container.querySelector('input[name="container_image"]') as HTMLInputElement | null)?.value).toBe("ghcr.io/acme/demo:latest")

    await act(async () => {
      root.unmount()
    })
  })
})
