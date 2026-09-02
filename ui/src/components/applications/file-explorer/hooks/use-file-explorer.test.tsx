import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook } from "@testing-library/react"
import type { PropsWithChildren } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { FileInfo, ReadFileResponse } from "@/api/file-explorer"

const { getHomeDirMock, listFilesMock, readFileMock } = vi.hoisted(() => ({
  getHomeDirMock: vi.fn(async () => ({ path: "/" })),
  listFilesMock: vi.fn(async () => ({ path: "/", files: [] })),
  readFileMock: vi.fn(),
}))

vi.mock("@/api/file-explorer", () => ({
  fileExplorerApi: {
    getHomeDir: getHomeDirMock,
    listFiles: listFilesMock,
    readFile: readFileMock,
  },
}))

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { useFileExplorer } from "./use-file-explorer"

function deferredRead() {
  let resolve: ((value: ReadFileResponse) => void) | undefined
  const promise = new Promise<ReadFileResponse>((promiseResolve) => {
    resolve = promiseResolve
  })
  return { promise, resolve: (value: ReadFileResponse) => resolve?.(value) }
}

describe("useFileExplorer", () => {
  beforeEach(() => {
    localStorage.clear()
    getHomeDirMock.mockClear()
    listFilesMock.mockClear()
    readFileMock.mockReset()
  })

  it("keeps the newest editor content when reads resolve out of order", async () => {
    const firstRead = deferredRead()
    const secondRead = deferredRead()
    readFileMock
      .mockReturnValueOnce(firstRead.promise)
      .mockReturnValueOnce(secondRead.promise)

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useFileExplorer({
      appId: "app-1",
      instanceName: "pod-1",
      containerName: "main",
    }), { wrapper })

    const firstFile: FileInfo = { name: "first.txt", type: "file", size: 1, modTime: 0, permissions: "0644" }
    const secondFile: FileInfo = { name: "second.txt", type: "file", size: 1, modTime: 0, permissions: "0644" }

    act(() => {
      result.current.handleOpen(firstFile)
      result.current.handleOpen(secondFile)
    })
    await act(async () => {
      secondRead.resolve({ path: "/second.txt", content: "second", size: 6 })
      await secondRead.promise
    })
    expect(result.current.editingFile?.path).toBe("/second.txt")
    expect(result.current.editingFile?.content).toBe("second")

    await act(async () => {
      firstRead.resolve({ path: "/first.txt", content: "first", size: 5 })
      await firstRead.promise
    })
    expect(result.current.editingFile?.path).toBe("/second.txt")
    expect(result.current.editingFile?.content).toBe("second")
  })
})
