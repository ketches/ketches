export function normalizeDomainValue(value: string) {
  return value.trim().toLowerCase()
}

export function isPatternDomain(value: string) {
  return value.startsWith("*.")
}

export function isValidDomainValue(value: string) {
  if (!value) {
    return true
  }

  return /^(?:\*\.)?[a-z0-9]([a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$/.test(value)
}

export function expandDomainPattern(slug: string, domainPattern?: string) {
  const normalizedSlug = slug.trim().toLowerCase()
  const normalizedDomainPattern = normalizeDomainValue(domainPattern ?? "")
  if (!normalizedSlug || !normalizedDomainPattern.startsWith("*.")) {
    return ""
  }

  return `${normalizedSlug}.${normalizedDomainPattern.slice(2)}`
}

export function seedDomainInputFromSelection(domain: string) {
  const normalizedDomain = normalizeDomainValue(domain)
  if (isPatternDomain(normalizedDomain)) {
    return normalizedDomain.slice(1)
  }
  return normalizedDomain
}
