import { describe, expect, it } from "vitest"

import { deriveExtensionDefaults, toExtensionSlug } from "./extension-dialog.utils"

describe("extension dialog utils", () => {
  it("derives name and slug from an OCI chart URL", () => {
    expect(
      deriveExtensionDefaults("oci://ghcr.io/nginx/charts/nginx-gateway-fabric")
    ).toEqual({
      name: "Nginx Gateway Fabric",
      slug: "nginx-gateway-fabric",
    })
  })

  it("slugifies manual names", () => {
    expect(toExtensionSlug("My Gateway API")).toBe("my-gateway-api")
  })
})
