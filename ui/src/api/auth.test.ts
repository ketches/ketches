import { beforeEach, describe, expect, it, vi } from "vitest"

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock("./client", () => ({
  default: {
    get: getMock,
    post: postMock,
  },
}))

import { authApi } from "./auth"

describe("authApi", () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it("posts credentials to the sign-in endpoint", async () => {
    postMock.mockResolvedValue({
      user: {
        id: "user-1",
        username: "admin",
        email: "admin@example.com",
        fullname: "Admin",
        role: "admin",
      },
      must_change_password: false,
      default_password_notice: "",
    })

    await expect(
      authApi.signIn({
        username: "admin",
        password: "secret",
      })
    ).resolves.toMatchObject({
      user: {
        username: "admin",
      },
    })

    expect(postMock).toHaveBeenCalledWith("/v1/users/sign-in", {
      username: "admin",
      password: "secret",
    })
  })

  it("loads the public sign-up configuration", async () => {
    getMock.mockResolvedValue({ enabled: true })

    await expect(authApi.getSignUpConfig()).resolves.toEqual({ enabled: true })

    expect(getMock).toHaveBeenCalledWith("/v1/users/sign-up/config")
  })
})
