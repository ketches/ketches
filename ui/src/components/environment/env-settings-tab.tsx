import { Loader2 } from "lucide-react"

import { EnvResourceQuotaCard } from "./env-resource-quota-card"

interface EnvSettingsTabProps {
  envId: string
  isViewer: boolean
}

export function EnvSettingsTab({ envId, isViewer }: EnvSettingsTabProps) {
  if (!envId) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <EnvResourceQuotaCard envId={envId} isViewer={isViewer} />
    </div>
  )
}
