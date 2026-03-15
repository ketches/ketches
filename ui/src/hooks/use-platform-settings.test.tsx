import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockGetBranding, mockUpdateBranding } = vi.hoisted(() => ({
  mockGetBranding: vi.fn(),
  mockUpdateBranding: vi.fn(),
}))

vi.mock("@/api/platform-settings", () => ({
  platformSettingsApi: {
    getBranding: (...args: unknown[]) => mockGetBranding(...args),
    updateBranding: (...args: unknown[]) => mockUpdateBranding(...args),
  },
}))

import { usePlatformBranding, useUpdatePlatformBrandingMutation } from "./use-platform-settings"

function createMemoryStorage(): Storage {
  const store = new Map<string, string>()

  return {
    get length() {
      return store.size
    },
    clear() {
      store.clear()
    },
    getItem(key) {
      return store.get(key) ?? null
    },
    key(index) {
      return Array.from(store.keys())[index] ?? null
    },
    removeItem(key) {
      store.delete(key)
    },
    setItem(key, value) {
      store.set(key, value)
    },
  }
}

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function BrandingReader() {
  const { data } = usePlatformBranding()

  return (
    <div>
      <span data-testid="brand-name">{data?.name ?? ""}</span>
    </div>
  )
}

function BrandingWriter() {
  const mutation = useUpdatePlatformBrandingMutation()

  return (
    <button
      type="button"
      onClick={() =>
        mutation.mutate({
          name: "Acme Control Plane",
        })
      }
    >
      Save
    </button>
  )
}

describe("usePlatformBranding", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("uses cached branding immediately before the server response arrives", async () => {
    localStorage.setItem("platform-branding", JSON.stringify({
      name: "Acme Control Plane",
    }))
    mockGetBranding.mockReturnValue(new Promise(() => undefined))

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <BrandingReader />
        </QueryClientProvider>
      )
      await flushPromises()
    })

    const brandName = container.querySelector('[data-testid="brand-name"]')

    expect(brandName?.textContent).toBe("Acme Control Plane")

    await act(async () => {
      root.unmount()
    })
  })

  it("persists the latest branding locally after a successful update", async () => {
    mockUpdateBranding.mockResolvedValue({
      name: "Acme Control Plane",
    })
    mockGetBranding.mockResolvedValue({
      name: "Acme Control Plane",
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <BrandingWriter />
        </QueryClientProvider>
      )
    })

    const saveButton = container.querySelector("button")

    await act(async () => {
      saveButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
      await flushPromises()
    })

    const cachedBranding = JSON.parse(localStorage.getItem("platform-branding") ?? "{}") as {
      name?: string
    }

    expect(cachedBranding.name).toBe("Acme Control Plane")
    expect(cachedBranding).toEqual({
      name: "Acme Control Plane",
    })

    await act(async () => {
      root.unmount()
    })
  })
})
