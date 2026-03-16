import { describe, expect, it } from "vitest"

import {
  deriveRepoDefaults,
  toRepositoryNameSlug,
} from "./code-repository-dialog.utils"

describe("code-repository-dialog.utils", () => {
  it("parses repo URLs and derives readable defaults", () => {
    expect(deriveRepoDefaults("https://github.com/acme/my-api.git")).toMatchObject({
      name: "My Api",
      slug: "my-api",
    })
    expect(deriveRepoDefaults("git@github.com:acme/my-api.git")).toMatchObject({
      name: "My Api",
      slug: "my-api",
    })
  })

  it("slugifies manual name edits with the existing rule", () => {
    expect(toRepositoryNameSlug("My API Service")).toBe("my-api-service")
  })
})
