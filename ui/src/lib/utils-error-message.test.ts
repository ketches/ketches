import { describe, expect, it } from "vitest"

import * as utils from "./utils"

function getErrorMessageImplementation() {
  return (utils as { getErrorMessage?: (error: unknown, fallback: string) => string }).getErrorMessage
}

describe("getErrorMessage", () => {
  it("prefers API response messages for axios-like errors", () => {
    const getErrorMessage = getErrorMessageImplementation()

    expect(getErrorMessage).toBeTypeOf("function")
    expect(
      getErrorMessage?.(
        {
          isAxiosError: true,
          message: "Request failed",
          response: { data: { error: "Server said no" } },
        },
        "Fallback message"
      )
    ).toBe("Server said no")
  })

  it("falls back to the axios error message when response data has no error", () => {
    const getErrorMessage = getErrorMessageImplementation()

    expect(getErrorMessage).toBeTypeOf("function")
    expect(
      getErrorMessage?.(
        {
          isAxiosError: true,
          message: "Network exploded",
          response: { data: {} },
        },
        "Fallback message"
      )
    ).toBe("Network exploded")
  })

  it("uses standard error messages for non-axios errors", () => {
    const getErrorMessage = getErrorMessageImplementation()

    expect(getErrorMessage).toBeTypeOf("function")
    expect(getErrorMessage?.(new Error("Plain failure"), "Fallback message")).toBe("Plain failure")
  })

  it("returns the fallback for unknown values", () => {
    const getErrorMessage = getErrorMessageImplementation()

    expect(getErrorMessage).toBeTypeOf("function")
    expect(getErrorMessage?.({ reason: "mystery" }, "Fallback message")).toBe("Fallback message")
  })
})
