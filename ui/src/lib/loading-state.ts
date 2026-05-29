export interface LoadingSourceState {
  isLoading?: boolean
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
    return Boolean(favoritesSource?.isLoading)
  }

  if (hasExternalDataSource) {
    return Boolean(externalSource?.isLoading)
  }

  return Boolean(defaultSource?.isLoading)
}
