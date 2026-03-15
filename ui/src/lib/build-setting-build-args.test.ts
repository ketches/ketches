import { describe, expect, it } from "vitest"

import {
  parseBuildArgs,
  serializeBuildArgPairs,
  validateBuildArgPairs,
} from "./build-setting-build-args"

describe("build setting build args helpers", () => {
  it("parses newline-delimited build args into sorted key/value rows", () => {
    expect(parseBuildArgs("ZETA=last\nALPHA=first")).toEqual({
      mode: "structured",
      pairs: [
        { key: "ALPHA", value: "first" },
        { key: "ZETA", value: "last" },
      ],
      raw: "ZETA=last\nALPHA=first",
    })
  })

  it("serializes structured rows back to newline-delimited text sorted by key", () => {
    expect(serializeBuildArgPairs([
      { key: "ZETA", value: "last" },
      { key: "ALPHA", value: "first" },
    ])).toBe("ALPHA=first\nZETA=last")
  })

  it("returns advanced mode when raw args cannot be losslessly represented as key/value rows", () => {
    expect(parseBuildArgs("ALPHA=first\nEXPORT_ONLY")).toEqual({
      mode: "advanced",
      pairs: [],
      raw: "ALPHA=first\nEXPORT_ONLY",
    })
  })

  it("rejects duplicate keys before submit", () => {
    expect(validateBuildArgPairs([
      { key: "ALPHA", value: "first" },
      { key: "ALPHA", value: "second" },
    ])).toContain("Duplicate")
  })
})
