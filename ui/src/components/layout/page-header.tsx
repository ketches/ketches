import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useBreadcrumbs } from "@/contexts/use-breadcrumbs"
import { useEffect } from "react"

export function PageHeader({ items }: { items: BreadcrumbItem[] }) {
  const { setBreadcrumbs } = useBreadcrumbs()

  useEffect(() => {
    setBreadcrumbs(Array.isArray(items) ? items : [])
  }, [items, setBreadcrumbs])

  return null
}
