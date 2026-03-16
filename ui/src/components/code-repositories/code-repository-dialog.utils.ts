import { toTitleCase } from "@/lib/utils"

export function toRepositoryNameSlug(name: string) {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
}

export function deriveRepoDefaults(gitRepoUrl: string): { name: string; slug: string } {
  const urlText = gitRepoUrl.trim()
  if (!urlText) return { name: "", slug: "" }

  try {
    let path = ""

    if (urlText.includes("@") && urlText.includes(":") && !urlText.startsWith("http") && !urlText.startsWith("ssh://")) {
      path = urlText.split(":").pop() || ""
    } else {
      const url = new URL(urlText.startsWith("http") || urlText.startsWith("ssh://") ? urlText : `https://${urlText}`)
      path = url.pathname
    }

    path = path.replace(/^\/+/, "").replace(/\/+$/, "")
    const segment = path.split("/").pop() ?? "repo"
    const rawName = segment.replace(/\.git$/i, "").trim() || "repo"
    const slug = rawName
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "")
      .slice(0, 128) || "repo"

    const name = toTitleCase(rawName.replace(/[._-]+/g, " ").trim())

    return { name, slug }
  } catch {
    return { name: "", slug: "" }
  }
}
