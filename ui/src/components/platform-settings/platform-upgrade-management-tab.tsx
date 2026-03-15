import { ArrowUpCircle, Loader2, Package, RefreshCw, Settings2 } from "lucide-react"
import * as React from "react"

import type { PlatformUpdateStatus } from "@/api/platform-update"
import { PlatformUpdateConfigDialog } from "@/components/platform-updates/platform-update-config-dialog"
import { PlatformUpdateRolloutDialog } from "@/components/platform-updates/platform-update-rollout-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { usePlatformUpdateConfigQuery } from "@/hooks/use-platform-update"

interface PlatformUpgradeManagementTabProps {
  status?: PlatformUpdateStatus
  isStatusLoading?: boolean
  isChecking?: boolean
  onCheckForUpdates?: () => void
}

export function PlatformUpgradeManagementTab({
  status,
  isStatusLoading = false,
  isChecking = false,
  onCheckForUpdates,
}: PlatformUpgradeManagementTabProps) {
  const { data: config, isLoading: configLoading } = usePlatformUpdateConfigQuery(true)
  const [configDialogOpen, setConfigDialogOpen] = React.useState(false)
  const [rolloutDialogOpen, setRolloutDialogOpen] = React.useState(false)

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings2 className="h-4 w-4" />
            Upgrade Management
          </CardTitle>
          <CardDescription>
            Configure platform rollout targets, manually check the latest versions, and trigger
            upgrades for the API and UI.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Current Version</p>
              <p className="text-sm">
                {isStatusLoading
                  ? "Loading current platform status..."
                  : `API ${status?.api.current_version || "unknown"} / UI ${status?.ui.current_version || "unknown"}`}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-muted-foreground">Recommended Version</p>
              <p className="text-sm font-medium">
                {status?.recommended_shared_version || "No update detected"}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-muted-foreground">Rollout Status</p>
              <p className="text-sm">
                {status?.can_rollout ? "Ready for rollout" : status?.rollout_blockers?.[0] || "Waiting for a manual check"}
              </p>
            </div>
          </div>

          <div className="space-y-3">
            <div className="rounded-lg border bg-background/80 p-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0 flex-1 space-y-3">
                  <div className="flex items-start justify-between gap-3 space-y-1">
                    <div className="space-y-1">
                      <h3 className="text-sm font-medium">Upgrade Target</h3>
                      <p className="text-xs text-muted-foreground">
                        Review the current API and UI image repositories and Kubernetes deployment targets.
                      </p>
                    </div>
                    <div className="ml-auto">
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setConfigDialogOpen(true)}
                        disabled={configLoading}
                      >
                        Configure
                      </Button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <TargetSummary
                      title="API"
                      imageRegistry={config?.api.image_repository}
                      namespace={config?.api.namespace}
                      deploymentName={config?.api.deployment_name}
                      containerName={config?.api.container_name}
                      isLoading={configLoading}
                    />
                    <TargetSummary
                      title="UI"
                      imageRegistry={config?.ui.image_repository}
                      namespace={config?.ui.namespace}
                      deploymentName={config?.ui.deployment_name}
                      containerName={config?.ui.container_name}
                      isLoading={configLoading}
                    />
                  </div>
                </div>


              </div>
            </div>

            <ActionBlock
              icon={RefreshCw}
              title="Check for Updates"
              description="Pull the latest available image tags for ketches-api and ketches-ui and compare them with the running platform version."
              actionLabel="Check Now"
              onAction={() => onCheckForUpdates?.()}
              disabled={isChecking}
              isLoading={isChecking}
            />

            <ActionBlock
              icon={ArrowUpCircle}
              title="Rolling Update"
              description="Open the rolling update dialog to choose a target version and submit a Kubernetes rolling update for the platform API and UI."
              actionLabel="Rolling Update"
              onAction={() => setRolloutDialogOpen(true)}
              disabled={!status?.recommended_shared_version}
            />
          </div>
        </CardContent>
      </Card >

      <PlatformUpdateConfigDialog
        open={configDialogOpen}
        onOpenChange={setConfigDialogOpen}
      />

      <PlatformUpdateRolloutDialog
        open={rolloutDialogOpen}
        onOpenChange={setRolloutDialogOpen}
      />
    </>
  )
}

function ActionBlock({
  icon: Icon,
  title,
  description,
  actionLabel,
  onAction,
  disabled = false,
  isLoading = false,
}: {
  icon: typeof Settings2
  title: string
  description: string
  actionLabel: string
  onAction: () => void
  disabled?: boolean
  isLoading?: boolean
}) {
  return (
    <div className="rounded-lg border bg-background/80 p-4">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h3 className="flex items-center gap-2 text-sm font-medium">
            <Icon className="h-4 w-4" />
            {title}
          </h3>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>

        <div className="shrink-0">
          <Button type="button" variant="outline" onClick={onAction} disabled={disabled}>
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
            {actionLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}

function TargetSummary({
  title,
  imageRegistry: imageRegistry,
  namespace,
  deploymentName,
  containerName,
  isLoading,
}: {
  title: string
  imageRegistry?: string
  namespace?: string
  deploymentName?: string
  containerName?: string
  isLoading: boolean
}) {
  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <div className="mb-2 flex items-center gap-2">
        <Package className="h-4 w-4 text-muted-foreground" />
        <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</h4>
      </div>
      <div className="space-y-2">
        <SummaryRow label="Image Registry" value={imageRegistry} isLoading={isLoading} />
        <div className="grid grid-cols-3">
          <SummaryRow label="Namespace" value={namespace} isLoading={isLoading} />
          <SummaryRow label="Deployment" value={deploymentName} isLoading={isLoading} />
          <SummaryRow label="Container" value={containerName} isLoading={isLoading} />
        </div>
      </div>
    </div>
  )
}

function SummaryRow({
  label,
  value,
  isLoading,
}: {
  label: string
  value?: string
  isLoading: boolean
}) {
  return (
    <div>
      <p className="text-[11px] font-medium text-muted-foreground">{label}</p>
      <p className="text-xs font-mono break-all">
        {isLoading ? "Loading..." : value || "Not configured"}
      </p>
    </div>
  )
}
