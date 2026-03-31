export const IMAGE_PULL_POLICY_OPTIONS = [
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
] as const

export const getImagePullPolicyLabel = (value: string | null | undefined) => {
  return IMAGE_PULL_POLICY_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""
}
