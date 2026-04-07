import { beforeEach, describe, expect, it, vi } from "vitest"

const { deleteMock, getMock, patchMock, postMock, putMock } = vi.hoisted(() => ({
  deleteMock: vi.fn(),
  getMock: vi.fn(),
  patchMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock("./client", () => ({
  default: {
    delete: deleteMock,
    get: getMock,
    patch: patchMock,
    post: postMock,
    put: putMock,
  },
}))

import { appsApi } from "./apps"

describe("appsApi", () => {
  beforeEach(() => {
    deleteMock.mockReset()
    getMock.mockReset()
    patchMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
  })

  it("passes pagination params when listing apps", async () => {
    getMock.mockResolvedValue({
      items: [],
      pagination: {
        total: 0,
        page: 1,
        page_size: 20,
        total_pages: 0,
      },
    })

    await expect(appsApi.list("env-1", { page: 1, page_size: 20, search: "api" })).resolves.toEqual({
      items: [],
      pagination: {
        total: 0,
        page: 1,
        page_size: 20,
        total_pages: 0,
      },
    })

    expect(getMock).toHaveBeenCalledWith("/v1/envs/env-1/apps", {
      params: {
        page: 1,
        page_size: 20,
        search: "api",
      },
    })
  })

  it("patches image settings to the app image endpoint", async () => {
    patchMock.mockResolvedValue({ id: "app-1" })

    await appsApi.updateImage("app-1", {
      container_image: "nginx:1.27",
      image_pull_policy: "IfNotPresent",
      registry_username: "robot",
      clear_registry_password: true,
    })

    expect(patchMock).toHaveBeenCalledWith("/v1/apps/app-1/image", {
      container_image: "nginx:1.27",
      image_pull_policy: "IfNotPresent",
      registry_username: "robot",
      clear_registry_password: true,
    })
  })

  it("loads topology resource yaml from the node endpoint", async () => {
    getMock.mockResolvedValue({ yaml: "kind: Deployment" })

    await expect(appsApi.getTopologyResourceYaml("app-1", "deploy-1")).resolves.toEqual({
      yaml: "kind: Deployment",
    })

    expect(getMock).toHaveBeenCalledWith("/v1/apps/app-1/topology/nodes/deploy-1/resource-yaml")
  })
})
