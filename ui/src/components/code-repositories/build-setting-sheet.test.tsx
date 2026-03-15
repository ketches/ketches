import { act, createContext, useContext, useMemo } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: ({
    checked,
    onCheckedChange,
    ...props
  }: {
    checked?: boolean
    onCheckedChange?: (checked: boolean) => void
  } & React.ComponentProps<"input">) => (
    <input
      {...props}
      checked={checked}
      type="checkbox"
      onChange={(event) => onCheckedChange?.(event.target.checked)}
    />
  ),
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children }: { children: React.ReactNode }) => <label>{children}</label>,
}))

const ComboboxContext = createContext<{ display: string }>({ display: "" })

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({
    children,
    value,
    itemToStringLabel,
  }: {
    children: React.ReactNode
    value: string
    itemToStringLabel: (value: string) => string
  }) => {
    const display = useMemo(() => itemToStringLabel(value), [itemToStringLabel, value])
    return <ComboboxContext.Provider value={{ display }}>{children}</ComboboxContext.Provider>
  },
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: (props: React.ComponentProps<"input">) => {
    const { display } = useContext(ComboboxContext)
    return <input {...props} readOnly value={display} />
  },
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/code-repositories/git-ref-select", () => ({
  GitRefSelect: ({ value, onValueChange }: { value: string; onValueChange: (value: string) => void }) => (
    <input
      data-testid="git-ref-select"
      value={value}
      onChange={(event) => onValueChange(event.target.value)}
    />
  ),
}))

vi.mock("@/components/shared/key-value-input", () => ({
  KeyValueInput: ({ value }: { value: { key: string; value: string }[] }) => (
    <div data-testid="key-value-input">{JSON.stringify(value)}</div>
  ),
}))

import type { BuildSetting, BuildArgPair } from "@/api/code-repositories"
import { BuildSettingSheet } from "./build-setting-sheet"

const registries = [
  { id: "registry-1", name: "Main Registry", provider: "ghcr" },
] as const

describe("BuildSettingSheet", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("defaults new build settings to linux/amd64 with registry cache enabled", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <BuildSettingSheet
          mode="create"
          open
          onOpenChange={() => undefined}
          repoId="repo-1"
          repoSlug="demo-api"
          registries={registries as never}
          onSubmit={() => undefined}
        />
      )
    })

    expect(container.textContent).toContain("linux/amd64")
    expect(container.textContent).toContain("linux/amd64,linux/arm64")
    expect((container.querySelector('input[type="checkbox"]') as HTMLInputElement | null)?.checked).toBe(true)
    expect((container.querySelector('input[placeholder="my-service"]') as HTMLInputElement | null)?.value).toBe("demo-api")

    await act(async () => {
      root.unmount()
    })
  })

  it("uses KeyValueInput in structured mode by default", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <BuildSettingSheet
          mode="create"
          open
          onOpenChange={() => undefined}
          repoId="repo-1"
          repoSlug="demo-api"
          registries={registries as never}
          onSubmit={() => undefined}
        />
      )
    })

    expect(container.querySelector('[data-testid="key-value-input"]')).not.toBeNull()
    expect(container.querySelector("textarea")).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("falls back to advanced mode when existing build_args are not key/value pairs", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    const setting = {
      id: "setting-1",
      code_repository_id: "repo-1",
      name: "backend",
      git_ref: "main",
      dockerfile_path: "Dockerfile",
      build_context: ".",
      image_name: "demo-api",
      registry_id: "registry-1",
      build_args: "ALPHA=first\nEXPORT_ONLY",
      platforms: "linux/amd64",
      registry_cache_enabled: true,
      registry_cache_ref: "",
    } satisfies BuildSetting

    await act(async () => {
      root.render(
        <BuildSettingSheet
          mode="edit"
          open
          onOpenChange={() => undefined}
          repoId="repo-1"
          repoSlug="demo-api"
          registries={registries as never}
          setting={setting}
          onSubmit={() => undefined}
        />
      )
    })

    expect(container.querySelector("textarea")).not.toBeNull()
    expect((container.querySelector("textarea") as HTMLTextAreaElement | null)?.value).toContain("EXPORT_ONLY")

    await act(async () => {
      root.unmount()
    })
  })

  it("submits platforms, registry cache settings, and serialized build args", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const onSubmit = vi.fn()

    const setting = {
      id: "setting-1",
      code_repository_id: "repo-1",
      name: "backend",
      git_ref: "main",
      dockerfile_path: "Dockerfile",
      build_context: ".",
      image_name: "demo-api",
      registry_id: "registry-1",
      build_args: "ALPHA=first\nZETA=last",
      build_arg_pairs: [
        { key: "ALPHA", value: "first" },
        { key: "ZETA", value: "last" },
      ] satisfies BuildArgPair[],
      platforms: "linux/amd64,linux/arm64",
      registry_cache_enabled: false,
      registry_cache_ref: "ghcr.io/demo/api:buildcache-setting-1",
    } satisfies BuildSetting

    await act(async () => {
      root.render(
        <BuildSettingSheet
          mode="edit"
          open
          onOpenChange={() => undefined}
          repoId="repo-1"
          repoSlug="demo-api"
          registries={registries as never}
          setting={setting}
          onSubmit={onSubmit}
        />
      )
    })

    const saveButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Save")
    )

    await act(async () => {
      saveButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
      platforms: "linux/amd64,linux/arm64",
      registry_cache_enabled: false,
      registry_cache_ref: "ghcr.io/demo/api:buildcache-setting-1",
      build_args: "ALPHA=first\nZETA=last",
    }))

    await act(async () => {
      root.unmount()
    })
  })
})
