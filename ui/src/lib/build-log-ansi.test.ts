import { describe, expect, it } from "vitest"

import { parseBuildLogAnsi } from "./build-log-ansi"

describe("parseBuildLogAnsi", () => {
  it("strips ANSI control sequences and returns Monaco-friendly color ranges", () => {
    const parsed = parseBuildLogAnsi("\u001b[36mINFO\u001b[0m [0032] Saving file")

    expect(parsed.text).toBe("INFO [0032] Saving file")
    expect(parsed.decorations).toEqual([
      {
        endColumn: 5,
        endLineNumber: 1,
        inlineClassName: "build-log-ansi-fg-cyan",
        startColumn: 1,
        startLineNumber: 1,
      },
    ])
  })

  it("keeps color active until reset across line breaks", () => {
    const parsed = parseBuildLogAnsi("\u001b[31mFAIL\nstill failing\u001b[0m")

    expect(parsed.text).toBe("FAIL\nstill failing")
    expect(parsed.decorations).toEqual([
      {
        endColumn: 5,
        endLineNumber: 1,
        inlineClassName: "build-log-ansi-fg-red",
        startColumn: 1,
        startLineNumber: 1,
      },
      {
        endColumn: 14,
        endLineNumber: 2,
        inlineClassName: "build-log-ansi-fg-red",
        startColumn: 1,
        startLineNumber: 2,
      },
    ])
  })
})
