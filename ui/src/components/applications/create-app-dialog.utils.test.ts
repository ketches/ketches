import { describe, expect, it } from "vitest"

import { deriveImageDefaults, isStatefulImage } from "./create-app-dialog.utils"

describe("create-app-dialog.utils", () => {
  it("extracts the image base name from tagged and digested images", () => {
    expect(deriveImageDefaults("ghcr.io/acme/postgres:16")).toMatchObject({
      imageName: "postgres",
      slug: "postgres",
      name: "Postgres",
    })
    expect(deriveImageDefaults("ghcr.io/acme/postgres@sha256:abc")).toMatchObject({
      imageName: "postgres",
      slug: "postgres",
      name: "Postgres",
    })
  })

  it("turns separators into a readable title-cased name", () => {
    expect(deriveImageDefaults("redis-stack-server:latest")).toMatchObject({
      imageName: "redis-stack-server",
      slug: "redis-stack-server",
      name: "Redis Stack Server",
    })
  })

  it("matches known stateful images", () => {
    expect(isStatefulImage("bitnami/mysql:8.0")).toBe(true)
    expect(isStatefulImage("nginx:latest")).toBe(false)
  })
})
