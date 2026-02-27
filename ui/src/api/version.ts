// Version API module for the platform version endpoint.
import client from './client'

export interface VersionInfo {
  version: string
  build_time: string
}

export const versionApi = {
  get: async () => {
    return client.get('/v1/version') as Promise<VersionInfo>
  },
}
