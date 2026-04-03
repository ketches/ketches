import * as React from "react"
import { useSearchParams } from "react-router-dom"

export type ApplicationInstancesViewMode = "table" | "card"

const INSTANCES_VIEW_MODE_KEY = "instances_view_mode"

function isInstancesViewMode(value: string): value is ApplicationInstancesViewMode {
  return value === "table" || value === "card"
}

export function useApplicationDetailTabs() {
  const [searchParams, setSearchParams] = useSearchParams()
  const currentTab = searchParams.get("tab") || "overview"
  const [viewMode, setViewModeState] = React.useState<ApplicationInstancesViewMode>(() => {
    const saved = localStorage.getItem(INSTANCES_VIEW_MODE_KEY)
    return isInstancesViewMode(saved ?? "") ? saved : "table"
  })

  React.useEffect(() => {
    localStorage.setItem(INSTANCES_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const setCurrentTab = React.useCallback((tab: string) => {
    setSearchParams({ tab }, { replace: true })
  }, [setSearchParams])

  const setViewMode = React.useCallback((value: string) => {
    if (!isInstancesViewMode(value)) {
      return
    }

    setViewModeState(value)
  }, [])

  return {
    currentTab,
    setCurrentTab,
    viewMode,
    setViewMode,
  }
}
