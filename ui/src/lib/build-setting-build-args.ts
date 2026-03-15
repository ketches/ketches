import type { BuildArgPair } from "@/api/code-repositories"

type BuildArgsParseResult = {
  mode: "structured" | "advanced"
  pairs: BuildArgPair[]
  raw: string
}

function normalizePair(pair: BuildArgPair): BuildArgPair {
  return {
    key: pair.key.trim(),
    value: pair.value.trim(),
  }
}

export function parseBuildArgs(raw: string): BuildArgsParseResult {
  const normalizedRaw = raw.trim()
  if (!normalizedRaw) {
    return { mode: "structured", pairs: [], raw: "" }
  }

  const lines = normalizedRaw.split("\n")
  const pairs: BuildArgPair[] = []
  const seen = new Set<string>()

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) continue

    const separatorIndex = trimmed.indexOf("=")
    if (separatorIndex <= 0) {
      return { mode: "advanced", pairs: [], raw: normalizedRaw }
    }

    const pair = normalizePair({
      key: trimmed.slice(0, separatorIndex),
      value: trimmed.slice(separatorIndex + 1),
    })
    if (!pair.key || seen.has(pair.key)) {
      return { mode: "advanced", pairs: [], raw: normalizedRaw }
    }

    seen.add(pair.key)
    pairs.push(pair)
  }

  return {
    mode: "structured",
    pairs: pairs.sort((a, b) => a.key.localeCompare(b.key)),
    raw: normalizedRaw,
  }
}

export function serializeBuildArgPairs(pairs: BuildArgPair[]): string {
  return pairs
    .map(normalizePair)
    .filter((pair) => pair.key)
    .sort((a, b) => a.key.localeCompare(b.key))
    .map((pair) => `${pair.key}=${pair.value}`)
    .join("\n")
}

export function validateBuildArgPairs(pairs: BuildArgPair[]): string | null {
  const seen = new Set<string>()

  for (const pair of pairs) {
    const normalized = normalizePair(pair)
    if (!normalized.key) {
      return "Build arg key is required."
    }
    if (seen.has(normalized.key)) {
      return `Duplicate build arg key: ${normalized.key}`
    }
    seen.add(normalized.key)
  }

  return null
}
