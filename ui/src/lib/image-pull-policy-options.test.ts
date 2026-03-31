import { describe, expect, it } from "vitest"

import { IMAGE_PULL_POLICY_OPTIONS } from "./image-pull-policy-options"

describe("IMAGE_PULL_POLICY_OPTIONS", () => {
  it("defines all supported pull policies with descriptions", () => {
    expect(IMAGE_PULL_POLICY_OPTIONS).toEqual([
      {
        label: "IfNotPresent",
        value: "IfNotPresent",
        description: "Pull the image only if it is not already present on the node. This is the default pull policy and is suitable for most cases.",
      },
      {
        label: "Always",
        value: "Always",
        description: "Always pull the image from the registry, regardless of whether it is already present on the node.",
      },
      {
        label: "Never",
        value: "Never",
        description: "Never pull the image from the registry. If the image is not present on the node, the pod will fail to start.",
      },
    ])
  })
})
