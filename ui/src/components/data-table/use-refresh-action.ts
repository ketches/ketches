import * as React from "react"

interface UseRefreshActionOptions {
  onRefresh?: (() => void | Promise<unknown>) | undefined
  isLoading?: boolean
}

export function useRefreshAction({ onRefresh, isLoading = false }: UseRefreshActionOptions) {
  const [isRefreshing, setIsRefreshing] = React.useState(false)

  const handleRefresh = React.useCallback(async () => {
    if (!onRefresh || isRefreshing) {
      return
    }

    try {
      setIsRefreshing(true)
      await Promise.resolve(onRefresh())
    } finally {
      setIsRefreshing(false)
    }
  }, [isRefreshing, onRefresh])

  return {
    isRefreshing,
    showRefreshOverlay: isRefreshing && !isLoading,
    handleRefresh,
  }
}
