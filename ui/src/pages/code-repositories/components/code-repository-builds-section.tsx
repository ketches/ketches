import { type ColumnDef } from "@tanstack/react-table"
import { Clock, Copy, FileClock, GitBranch, Loader2, RotateCw, Rocket } from "lucide-react"
import { toast } from "sonner"

import type { Build } from "@/api/builds"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import type { RetryBuildPayload } from "@/pages/code-repositories/hooks/use-code-repository-detail"
import { formatDate } from "@/lib/utils"

interface CodeRepositoryBuildsSectionProps {
  builds: Build[]
  isLoading: boolean
  isViewer: boolean
  retryingBuildId: string | null
  isRetryPending: boolean
  settingNameById: (id: string) => string
  onViewLogs: (buildId: string) => void
  onRetry: (payload: RetryBuildPayload) => void
  onDeploy: (buildId: string, buildSettingId?: string) => void
}

function formatDuration(seconds: number): string {
  return seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

export function CodeRepositoryBuildsSection({
  builds,
  isLoading,
  isViewer,
  retryingBuildId,
  isRetryPending,
  settingNameById,
  onViewLogs,
  onRetry,
  onDeploy,
}: CodeRepositoryBuildsSectionProps) {
  const columns: ColumnDef<Build>[] = [
    {
      accessorKey: "build_number",
      header: "#",
      cell: ({ row }) => row.original.build_number,
    },
    {
      id: "setting",
      header: "Build Setting",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.build_setting_id ? settingNameById(row.original.build_setting_id) : "-"}
        </span>
      ),
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <GitBranch className="h-3 w-3" />
          {row.original.git_ref}
        </span>
      ),
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="block truncate font-mono text-xs">{row.original.image_full_name}</span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 transition-opacity group-hover/row:opacity-100"
            onClick={(event) => {
              event.stopPropagation()
              navigator.clipboard.writeText(row.original.image_full_name)
              toast.success("Image address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "duration",
      header: "Duration",
      cell: ({ row }) => row.original.duration ? formatDuration(row.original.duration) : "-",
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <BuildStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at)}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <span className="flex justify-end">Actions</span>,
      cell: ({ row }) => {
        const build = row.original
        const isRetryingCurrentBuild = isRetryPending && retryingBuildId === build.id

        return (
          <div className="flex items-center justify-end gap-1">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={<Button variant="ghost" size="icon-sm" onClick={() => onViewLogs(build.id)} />}
              >
                <FileClock />
              </TooltipTrigger>
              <TooltipContent>View Build Logs</TooltipContent>
            </Tooltip>
            {!isViewer && (build.status === "failed" || build.status === "cancelled") && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onRetry({
                        id: build.id,
                        build_setting_id: build.build_setting_id,
                        build_env_id: build.build_env_id,
                        git_ref: build.git_ref,
                      })}
                      disabled={isRetryingCurrentBuild}
                    />
                  }
                >
                  <>
                    {isRetryingCurrentBuild ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCw />
                    )}
                    Retry
                  </>
                </TooltipTrigger>
                <TooltipContent>Retry Build</TooltipContent>
              </Tooltip>
            )}
            {!isViewer && build.status === "succeeded" && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="outline" size="sm" onClick={() => onDeploy(build.id, build.build_setting_id)} />}
                >
                  <>
                    <Rocket />
                    Deploy
                  </>
                </TooltipTrigger>
              </Tooltip>
            )}
          </div>
        )
      },
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <FileClock className="h-4 w-4" />
          Build History
        </CardTitle>
        <CardDescription>
          All builds for this repository. Deploy succeeded builds to an environment.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          data={builds}
          sourceDataCount={builds.length}
          isLoading={isLoading}
          sourceEmptyContent={(
            <EmptyState
              title="No builds yet"
              description="Trigger a build from a configuration above to see the history here."
              icon={FileClock}
            />
          )}
          useStandaloneEmptyState
        />
      </CardContent>
    </Card>
  )
}
