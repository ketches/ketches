import React, { createContext, type ReactNode } from "react"

export interface BreadcrumbItem {
    label: string
    href?: string
    icon?: React.ElementType
    dropdown?: ReactNode
}

export interface BreadcrumbContextType {
    breadcrumbs: BreadcrumbItem[]
    setBreadcrumbs: (items: BreadcrumbItem[]) => void
}

export const BreadcrumbContext = createContext<BreadcrumbContextType | undefined>(undefined)
