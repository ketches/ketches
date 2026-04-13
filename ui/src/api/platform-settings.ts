import client from "./client"

export interface PlatformBranding {
  name: string
}

export interface UpdatePlatformBrandingRequest {
  name: string
}

export interface PublicSignUpSettings {
  enabled: boolean
  email_verification_required: boolean
}

export const platformSettingsApi = {
  getBranding: async () => {
    return client.get("/v1/platform-settings/branding") as Promise<PlatformBranding>
  },

  updateBranding: async (data: UpdatePlatformBrandingRequest) => {
    return client.put("/v1/platform-settings/branding", data) as Promise<PlatformBranding>
  },
  getPublicSignUpSettings: async () => {
    return client.get("/v1/platform-settings/public-sign-up") as Promise<PublicSignUpSettings>
  },
  updatePublicSignUpSettings: async (data: PublicSignUpSettings) => {
    return client.put("/v1/platform-settings/public-sign-up", data) as Promise<PublicSignUpSettings>
  },
}
