export interface LoadingSourceState {
  isLoading?: boolean
  isFetching?: boolean
}

export interface SelectLoadingStateOptions {
  favoritesOnly?: boolean
  hasExternalDataSource?: boolean
  favoritesSource?: LoadingSourceState
  externalSource?: LoadingSourceState
  defaultSource?: LoadingSourceState
}

export function selectLoadingState({
  favoritesOnly = false,
  hasExternalDataSource = false,
  favoritesSource,
  externalSource,
  defaultSource,
}: SelectLoadingStateOptions): boolean {
  if (favoritesOnly) {
    return Boolean(favoritesSource?.isLoading || favoritesSource?.isFetching)
  }

  if (hasExternalDataSource) {
    return Boolean(externalSource?.isLoading || externalSource?.isFetching)
  }

  return Boolean(defaultSource?.isLoading || defaultSource?.isFetching)
}
