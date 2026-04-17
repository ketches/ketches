const MAX_HELM_RELEASE_NAME_LENGTH = 53

export function sanitizeHelmReleaseName(input: string): string {
  const normalized = input
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+/, "")

  const truncated = normalized.slice(0, MAX_HELM_RELEASE_NAME_LENGTH)
  return truncated.replace(/-+$/, "")
}
