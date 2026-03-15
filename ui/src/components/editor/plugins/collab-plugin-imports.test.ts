import { describe, expect, it } from "vitest"

import { CollabAutoformatKit } from "./collab-autoformat-kit"
import { CollabBlocksKit } from "./collab-blocks-kit"
import { CollabColorKit } from "./collab-color-kit"
import { CollabLinkKit } from "./collab-link-kit"
import { CollabListsKit } from "./collab-lists-kit"

describe("collab editor plugin imports", () => {
  it("resolves all collab editor helper modules", () => {
    expect(CollabAutoformatKit).toBeTruthy()
    expect(CollabBlocksKit).toBeTruthy()
    expect(CollabColorKit).toBeTruthy()
    expect(CollabLinkKit).toBeTruthy()
    expect(CollabListsKit).toBeTruthy()
  })
})
