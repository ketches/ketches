import { toTitleCase } from "@/lib/utils"

export function toExtensionSlug(name: string) {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 128)
}

export function deriveExtensionDefaults(ociUrl: string): { name: string; slug: string } {
  const value = ociUrl.trim()
  if (!value) return { name: "", slug: "" }

  const withoutScheme = value.replace(/^oci:\/\//i, "")
  const withoutDigest = withoutScheme.split("@")[0] ?? ""
  const lastSegment = withoutDigest.replace(/\/+$/, "").split("/").pop() ?? ""
  const rawName = (lastSegment.split(":")[0] ?? "").trim()
  const slug = toExtensionSlug(rawName)
  const name = toTitleCase(rawName.replace(/[._-]+/g, " ").replace(/\s+/g, " ").trim())

  return { name, slug }
}
