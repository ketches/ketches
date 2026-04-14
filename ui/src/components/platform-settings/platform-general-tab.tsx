import { PlatformBasicSettingsCard } from "@/components/platform-settings/platform-basic-settings-card"
import { PlatformBrandingTab } from "@/components/platform-settings/platform-branding-tab"
import { PlatformOperationLogRetentionCard } from "@/components/platform-settings/platform-operation-log-retention-card"

export function PlatformGeneralTab() {
  return (
    <div className="space-y-4">
      <PlatformBasicSettingsCard />
      <PlatformBrandingTab />
      <PlatformOperationLogRetentionCard />
    </div>
  )
}
