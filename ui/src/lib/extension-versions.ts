import type { ExtensionVersionInfo } from "@/api/clusters"

export function filterSelectableExtensionVersions(
  versions: ExtensionVersionInfo[],
  currentVersion?: string,
): ExtensionVersionInfo[] {
  const filtered = versions.filter((item) => !item.version.startsWith("sha256-"))

  if (
    currentVersion &&
    !currentVersion.startsWith("sha256-") &&
    !filtered.some((item) => item.version === currentVersion)
  ) {
    return [{ version: currentVersion }, ...filtered]
  }

  return filtered
}
