// useVersion returns the platform version string.
// Returns null while loading or on error.
import { useQuery } from '@tanstack/react-query'
import { versionApi } from '@/api/version'

export function useVersion(): string | null {
  const { data } = useQuery({
    queryKey: ['platform-version'],
    queryFn: versionApi.get,
    staleTime: 10 * 60 * 1000, // Version rarely changes — cache 10 minutes
  })
  return data?.version ?? null
}
