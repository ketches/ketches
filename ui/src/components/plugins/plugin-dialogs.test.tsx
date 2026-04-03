import { act, createContext, useContext } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries, mockMutate, mockOnOpenChange, mockToastError } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockMutate: vi.fn(),
  mockOnOpenChange: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/api/plugins", () => ({
  pluginsApi: {
    createPlugin: vi.fn(),
    updatePlugin: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: mockToastError,
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

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
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

vi.mock("@/components/shared/key-value-input", () => ({
  KeyValueInput: () => <div>Environment Variables</div>,
}))

vi.mock("./../ui/item", () => ({
  Item: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ItemTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import { CreatePluginDialog } from "./create-plugin-dialog"
import { EditPluginDialog } from "./edit-plugin-dialog"

type PluginRecord = {
  id: string
  name: string
  slug: string
  description: string
  image: string
  image_pull_policy?: string
  registry_username?: string
  has_registry_password?: boolean
  command?: string
  plugin_type?: "init" | "sidecar"
  env_vars?: Array<{ key: string, value: string }>
}

async function renderCreateDialog() {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <CreatePluginDialog
        open
        onOpenChange={mockOnOpenChange}
        projectId="project-1"
      />
    )
  })

  return { container, root }
}

async function renderEditDialog(plugin: PluginRecord) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <EditPluginDialog
        open
        onOpenChange={mockOnOpenChange}
        projectId="project-1"
        plugin={plugin}
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

const changeInputValue = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

function buildPlugin(overrides: Partial<PluginRecord> = {}): PluginRecord {
  return {
    id: "plugin-1",
    name: "Migration",
    slug: "migration",
    description: "",
    image: "docker.io/library/migrate:latest",
    image_pull_policy: "IfNotPresent",
    registry_username: "",
    command: "",
    plugin_type: "init",
    env_vars: [],
    ...overrides,
  }
}

describe("Plugin dialogs", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("keeps pull policy and registry credentials collapsed by default in create dialog", async () => {
    const { container, root } = await renderCreateDialog()

    expect(container.querySelector('input[name="image_pull_policy"]')).toBeNull()
    expect(container.querySelector("#registry_username")).toBeNull()
    expect(container.querySelector("#registry_password")).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("reveals pull policy and registry credentials when their buttons are clicked in create dialog", async () => {
    const { container, root } = await renderCreateDialog()

    const pullPolicyToggle = container.querySelector('button[aria-label="Pull Policy"]') as HTMLButtonElement | null
    const credentialsToggle = container.querySelector('button[aria-label="Registry credentials"]') as HTMLButtonElement | null

    expect(pullPolicyToggle).not.toBeNull()
    expect(credentialsToggle).not.toBeNull()

    await clickElement(pullPolicyToggle as HTMLButtonElement)
    await clickElement(credentialsToggle as HTMLButtonElement)

    expect(container.textContent).toContain("Pull Policy")
    expect(container.textContent).toContain("Registry Username")
    expect(container.textContent).toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("shows pull policy immediately in edit dialog when the policy is not IfNotPresent", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      image_pull_policy: "Always",
    }))

    expect(container.textContent).toContain("Pull Policy")

    await act(async () => {
      root.unmount()
    })
  })

  it("shows registry credentials immediately in edit dialog when registry username exists", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      registry_username: "robot",
    }))

    expect(container.textContent).toContain("Registry Username")
    expect(container.textContent).toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("prevents create submit when password is provided without a registry username", async () => {
    const { container, root } = await renderCreateDialog()

    const credentialsToggle = container.querySelector('button[aria-label="Registry credentials"]') as HTMLButtonElement | null
    const nameInput = container.querySelector("#name") as HTMLInputElement | null
    const slugInput = container.querySelector("#slug") as HTMLInputElement | null
    const imageInput = container.querySelector("#image") as HTMLInputElement | null

    expect(credentialsToggle).not.toBeNull()
    expect(nameInput).not.toBeNull()
    expect(slugInput).not.toBeNull()
    expect(imageInput).not.toBeNull()

    await clickElement(credentialsToggle as HTMLButtonElement)
    await changeInputValue(nameInput as HTMLInputElement, "Registry Plugin")
    await changeInputValue(slugInput as HTMLInputElement, "registry-plugin")
    await changeInputValue(imageInput as HTMLInputElement, "docker.io/library/migrate:latest")

    const passwordInput = container.querySelector("#registry_password") as HTMLInputElement | null

    expect(passwordInput).not.toBeNull()

    await changeInputValue(passwordInput as HTMLInputElement, "secret")
    await clickElement(container.querySelector('button[type="submit"]') as HTMLButtonElement)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalledWith("Registry username is required when password is provided")

    await act(async () => {
      root.unmount()
    })
  })

  it("prevents edit submit when password is provided without a registry username", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      registry_username: "robot",
    }))

    const usernameInput = container.querySelector("#edit-registry_username") as HTMLInputElement | null
    const passwordInput = container.querySelector("#edit-registry_password") as HTMLInputElement | null

    expect(usernameInput).not.toBeNull()
    expect(passwordInput).not.toBeNull()

    await changeInputValue(usernameInput as HTMLInputElement, "")
    await changeInputValue(passwordInput as HTMLInputElement, "secret")
    await clickElement(container.querySelector('button[type="submit"]') as HTMLButtonElement)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalledWith("Registry username is required when password is provided")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not restore the old registry username after it is cleared and password is edited", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      registry_username: "robot",
    }))

    const usernameInput = container.querySelector("#edit-registry_username") as HTMLInputElement | null
    const passwordInput = container.querySelector("#edit-registry_password") as HTMLInputElement | null

    expect(usernameInput).not.toBeNull()
    expect(passwordInput).not.toBeNull()

    await changeInputValue(usernameInput as HTMLInputElement, "")
    await changeInputValue(passwordInput as HTMLInputElement, "secret")

    expect((container.querySelector("#edit-registry_username") as HTMLInputElement | null)?.value).toBe("")

    await clickElement(container.querySelector('button[type="submit"]') as HTMLButtonElement)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalledWith("Registry username is required when password is provided")

    await act(async () => {
      root.unmount()
    })
  })

  it("sends an empty registry username on edit submit so credentials can be cleared", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      registry_username: "robot",
    }))

    const usernameInput = container.querySelector("#edit-registry_username") as HTMLInputElement | null

    expect(usernameInput).not.toBeNull()

    await changeInputValue(usernameInput as HTMLInputElement, "")
    await clickElement(container.querySelector('button[type="submit"]') as HTMLButtonElement)

    expect(mockMutate).toHaveBeenCalledWith(expect.objectContaining({
      registry_username: "",
    }))

    await act(async () => {
      root.unmount()
    })
  })

  it("shows a clear password icon on edit and restores an editable password input after clicking it", async () => {
    const { container, root } = await renderEditDialog(buildPlugin({
      registry_username: "robot",
      has_registry_password: true,
    }))

    const clearButton = container.querySelector('button[aria-label="Clear password"]') as HTMLButtonElement | null
    expect(clearButton).not.toBeNull()
    expect(clearButton?.textContent).toBe("")

    const passwordInputBefore = container.querySelector("#edit-registry_password") as HTMLInputElement | null
    expect(passwordInputBefore).not.toBeNull()
    expect(passwordInputBefore?.readOnly).toBe(true)

    await clickElement(clearButton as HTMLButtonElement)

    const passwordInputAfter = container.querySelector("#edit-registry_password") as HTMLInputElement | null
    expect(passwordInputAfter).not.toBeNull()
    expect(passwordInputAfter?.readOnly).toBe(false)
    expect(passwordInputAfter?.disabled).toBe(false)
    expect(container.querySelector('button[aria-label="Clear password"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
