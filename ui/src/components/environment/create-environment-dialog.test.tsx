import { act } from "react"
import ReactDOMClient from "react-dom/client"
import * as React from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockMutate,
  mockInvalidateQueries,
  mockOnOpenChange,
} = vi.hoisted(() => ({
  mockMutate: vi.fn(),
  mockInvalidateQueries: vi.fn(),
  mockOnOpenChange: vi.fn(),
}))

const CLUSTERS = [
  {
    id: "cluster-1",
    name: "Cluster One",
    slug: "cluster-one",
    connection_status: "connected",
  },
  {
    id: "cluster-2",
    name: "Cluster Two",
    slug: "cluster-two",
    connection_status: "connected",
  },
]

const PROJECT = {
  id: "project-1",
  slug: "demo-project",
  name: "Demo Project",
}

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "clusters-public") {
      return {
        data: CLUSTERS,
        isLoading: false,
      }
    }

    if (queryKey[0] === "project") {
      return {
        data: PROJECT,
        isLoading: false,
      }
    }

    if (queryKey[0] === "cluster-namespaces") {
      return {
        data: queryKey[1] === "cluster-1" ? ["demo-project-production"] : [],
        isLoading: false,
      }
    }

    if (queryKey[0] === "env-namespace-availability") {
      const clusterId = queryKey[2]
      const namespace = queryKey[3]

      if (clusterId === "cluster-1" && namespace === "demo-project-production") {
        return {
          data: {
            available: false,
            source: "cluster",
            message: 'Namespace "demo-project-production" already exists in the selected cluster',
          },
          isLoading: false,
        }
      }

      if (namespace === "taken-db-ns") {
        return {
          data: {
            available: false,
            source: "database",
            message: 'Namespace "taken-db-ns" is already used by another environment in this cluster',
          },
          isLoading: false,
        }
      }

      return {
        data: {
          available: true,
          source: "available",
          message: `Namespace "${String(namespace)}" is available`,
        },
        isLoading: false,
      }
    }

    return {
      data: undefined,
      isLoading: false,
    }
  },
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    activeProjectId: "project-1",
  }),
}))

vi.mock("@/api/clusters", () => ({
  clustersApi: {
    listPublic: vi.fn(),
    listNamespaces: vi.fn(),
  },
}))

vi.mock("@/api/projects", () => ({
  projectsApi: {
    get: vi.fn(),
  },
}))

vi.mock("@/api/envs", () => ({
  envsApi: {
    create: vi.fn(),
    checkNamespaceAvailability: vi.fn(),
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
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => (
    <button type={type ?? "button"} {...props}>
      {children}
    </button>
  ),
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldError: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({
    children,
    htmlFor,
  }: {
    children: React.ReactNode
    htmlFor?: string
  }) => <label htmlFor={htmlFor}>{children}</label>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({
    render,
    children,
    ...props
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  } & React.ComponentProps<"button">) => (
    <button type="button" {...props}>
      {render ?? children}
    </button>
  ),
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/item", () => ({
  Item: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  ItemContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  ItemDescription: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  ItemTitle: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock("@/components/ui/combobox", async () => {
  const React = await import("react")

  const ComboboxInput = () => null
  const ComboboxContent = ({ children }: { children: React.ReactNode }) => <>{children}</>
  const ComboboxList = ({ children }: { children: React.ReactNode }) => <>{children}</>
  const ComboboxItem = ({
    children,
  }: {
    children: React.ReactNode
    value: string
    disabled?: boolean
  }) => <>{children}</>

  const getText = (node: React.ReactNode): string => {
    if (typeof node === "string" || typeof node === "number") {
      return String(node)
    }

    if (Array.isArray(node)) {
      return node.map(getText).join("")
    }

    if (React.isValidElement(node)) {
      const element = node as React.ReactElement<{ children?: React.ReactNode }>
      return getText(element.props.children)
    }

    return ""
  }

  const collectItems = (children: React.ReactNode): Array<{ value: string; disabled?: boolean; label: string }> => {
    const items: Array<{ value: string; disabled?: boolean; label: string }> = []

    React.Children.forEach(children, (child) => {
      if (!React.isValidElement(child)) {
        return
      }

      const element = child as React.ReactElement<{
        children?: React.ReactNode
        value: string
        disabled?: boolean
      }>

      if (element.type === ComboboxItem) {
        items.push({
          value: element.props.value,
          disabled: element.props.disabled,
          label: getText(element.props.children) || element.props.value,
        })
        return
      }

      if (element.props.children) {
        items.push(...collectItems(element.props.children))
      }
    })

    return items
  }

  const Combobox = ({
    value,
    onValueChange,
    children,
  }: {
    value: string
    onValueChange: (value: string | null) => void
    children: React.ReactNode
  }) => {
    const items = collectItems(children)

    return (
      <select
        name="cluster_id"
        value={value}
        onChange={(event) => onValueChange(event.target.value || null)}
      >
        <option value="">Select a cluster</option>
        {items.map((item) => (
          <option key={item.value} value={item.value} disabled={item.disabled}>
            {item.label}
          </option>
        ))}
      </select>
    )
  }

  return {
    Combobox,
    ComboboxContent,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
  }
})

import { CreateEnvironmentDialog } from "./create-environment-dialog"

async function renderDialog() {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <CreateEnvironmentDialog
        open
        onOpenChange={mockOnOpenChange}
      />
    )
  })

  return { container, root }
}

const setInputValue = async (input: HTMLInputElement | HTMLTextAreaElement, value: string) => {
  await act(async () => {
    const proto = input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
    const valueSetter = Object.getOwnPropertyDescriptor(proto, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

const setSelectValue = async (select: HTMLSelectElement, value: string) => {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value")?.set
    valueSetter?.call(select, value)
    select.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

const clickElement = async (element: HTMLElement) => {
  await act(async () => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

describe("CreateEnvironmentDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows the namespace field only after both slug and cluster are selected", async () => {
    const { container, root } = await renderDialog()

    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null
    const clusterSelect = container.querySelector('select[name="cluster_id"]') as HTMLSelectElement | null

    expect(slugInput).not.toBeNull()
    expect(clusterSelect).not.toBeNull()
    expect(container.querySelector('input[name="namespace"]')).toBeNull()

    await setInputValue(slugInput as HTMLInputElement, "production")
    expect(container.querySelector('input[name="namespace"]')).toBeNull()

    await setSelectValue(clusterSelect as HTMLSelectElement, "cluster-1")
    expect((container.querySelector('input[name="namespace"]') as HTMLInputElement | null)?.value).toBe("demo-project-production")

    await act(async () => {
      root.unmount()
    })
  })

  it("updates namespace availability status in the label as the user edits values", async () => {
    const { container, root } = await renderDialog()

    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null
    const clusterSelect = container.querySelector('select[name="cluster_id"]') as HTMLSelectElement | null
    expect(slugInput).not.toBeNull()
    expect(clusterSelect).not.toBeNull()

    await setInputValue(slugInput as HTMLInputElement, "production")
    await setSelectValue(clusterSelect as HTMLSelectElement, "cluster-1")

    const namespaceLabel = container.querySelector('label[for="namespace"]')
    const namespaceInput = container.querySelector('input[name="namespace"]') as HTMLInputElement | null

    expect(namespaceLabel?.textContent).toContain("already exists in the selected cluster")
    expect(namespaceInput).not.toBeNull()

    await setInputValue(namespaceInput as HTMLInputElement, "shared-production")
    expect(container.querySelector('label[for="namespace"]')?.textContent).toContain('Namespace "shared-production" is available')

    await setInputValue(namespaceInput as HTMLInputElement, "taken-db-ns")
    expect(container.querySelector('label[for="namespace"]')?.textContent).toContain("already used by another environment")

    await act(async () => {
      root.unmount()
    })
  })

  it("regenerates namespace from the rule when the cluster changes", async () => {
    const { container, root } = await renderDialog()

    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null
    const clusterSelect = container.querySelector('select[name="cluster_id"]') as HTMLSelectElement | null

    expect(slugInput).not.toBeNull()
    expect(clusterSelect).not.toBeNull()

    await setInputValue(slugInput as HTMLInputElement, "production")
    await setSelectValue(clusterSelect as HTMLSelectElement, "cluster-1")

    const namespaceInput = container.querySelector('input[name="namespace"]') as HTMLInputElement | null
    expect(namespaceInput).not.toBeNull()

    await setInputValue(namespaceInput as HTMLInputElement, "shared-production")
    expect(namespaceInput?.value).toBe("shared-production")

    await setSelectValue(clusterSelect as HTMLSelectElement, "cluster-2")

    expect((container.querySelector('input[name="namespace"]') as HTMLInputElement | null)?.value).toBe("demo-project-production")
    expect(container.querySelector('label[for="namespace"]')?.textContent).toContain('Namespace "demo-project-production" is available')

    await act(async () => {
      root.unmount()
    })
  })

  it("blocks submission when namespace availability is not available", async () => {
    const { container, root } = await renderDialog()

    const nameInput = container.querySelector('input[name="name"]') as HTMLInputElement | null
    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null
    const clusterSelect = container.querySelector('select[name="cluster_id"]') as HTMLSelectElement | null
    const submitButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Create"),
    ) as HTMLButtonElement | undefined

    expect(nameInput).not.toBeNull()
    expect(slugInput).not.toBeNull()
    expect(clusterSelect).not.toBeNull()
    expect(submitButton).toBeDefined()

    await setInputValue(nameInput as HTMLInputElement, "Production")
    await setInputValue(slugInput as HTMLInputElement, "production")
    await setSelectValue(clusterSelect as HTMLSelectElement, "cluster-1")
    await clickElement(submitButton as HTMLButtonElement)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(container.querySelector('label[for="namespace"]')?.textContent).toContain("already exists in the selected cluster")

    await act(async () => {
      root.unmount()
    })
  })
})
