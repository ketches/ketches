import { useEffect } from "react"
import { useBreadcrumbs, type BreadcrumbItem } from "@/contexts/breadcrumb-context"

export function PageHeader({ items }: { items: BreadcrumbItem[] }) {
  const { setBreadcrumbs } = useBreadcrumbs()

  useEffect(() => {
    setBreadcrumbs(Array.isArray(items) ? items : [])
    return () => setBreadcrumbs([])
  }, [items, setBreadcrumbs])

  return null
}
