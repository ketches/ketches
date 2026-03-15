import client from "./client"

export interface PlatformBranding {
  name: string
}

export interface UpdatePlatformBrandingRequest {
  name: string
}

export const platformSettingsApi = {
  getBranding: async () => {
    return client.get("/v1/platform-settings/branding") as Promise<PlatformBranding>
  },

  updateBranding: async (data: UpdatePlatformBrandingRequest) => {
    return client.put("/v1/platform-settings/branding", data) as Promise<PlatformBranding>
  },
}
