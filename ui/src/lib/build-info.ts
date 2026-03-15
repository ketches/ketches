export function getAppBuildVersion(): string {
  return import.meta.env.VITE_APP_VERSION || "dev"
}

export function getAppBuildTime(): string {
  return import.meta.env.VITE_BUILD_TIME || ""
}
