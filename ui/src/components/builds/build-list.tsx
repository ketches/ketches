import { useQuery } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, Copy, FileClock, FolderGit2, Loader2, Package, User } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi } from "@/api/apps"
import { buildsApi, type Build } from "@/api/builds"
import { BuildLogViewer } from "@/components/builds/build-log-viewer"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { formatDate } from "@/lib/utils"

interface BuildListProps {
  appId: string
}

export function BuildList({ appId }: BuildListProps) {
  const [showLogDialog, setShowLogDialog] = React.useState<string | null>(null)
  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildId] = React.useState<string | undefined>(undefined)
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: app } = useQuery({
    queryKey: ['app', appId],
    queryFn: () => appsApi.get(appId),
  })

  const { data: buildsResponse, isLoading, refetch } = useQuery({
    queryKey: ['builds', appId, pagination.pageIndex + 1, pagination.pageSize],
    queryFn: () => buildsApi.list(appId, pagination.pageIndex + 1, pagination.pageSize),
    refetchInterval: 5000,
  })
  const builds = buildsResponse?.items ?? []
  const totalCount = buildsResponse?.pagination.total || 0


  const repoId = app?.code_repository_id
  const projectId = app?.env?.project_id

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}m ${s}s`
  }

  const columns: ColumnDef<Build>[] = [
    {
      id: "build_number",
      header: "#",
      cell: ({ row }) => (
        <span className="flex items-center gap-1 text-muted-foreground">
          {row.original.build_number}
        </span>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <BuildStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "git_ref",
      header: "Git Ref",
      cell: ({ row }) => (
        <span className="flex items-center text-muted-foreground gap-1 text-sm">
          <FolderGit2 className="h-3 w-3" />{row.original.git_ref}
        </span>
      ),
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono truncate block">
            {row.original.image_full_name || '-'}
          </span>
          <Button variant="ghost" size="icon-sm" className="opacity-0 group-hover/card:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              if (row.original.image_full_name) {
                navigator.clipboard.writeText(row.original.image_full_name)
                toast.success('Image name copied to clipboard')
              }
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
      cell: ({ row }) => {
        const build = row.original
        if (build.duration > 0) {
          return (
            <span className="flex items-center gap-1 text-xs">
              <Clock className="h-3 w-3" />{formatDuration(build.duration)}
            </span>
          )
        }
        if (build.status === 'building' || build.status === 'cloning') {
          return <Loader2 className="h-3 w-3 animate-spin" />
        }
        return '-'
      },
    },
    {
      accessorKey: "trigger_type",
      header: "Trigger",
      cell: ({ row }) => (
        <span className="flex items-center gap-1 text-xs capitalize">
          <User className="h-3 w-3" />{row.original.trigger_type}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground text-xs">
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
        return (
          <div
            className="flex items-center gap-1 justify-end"
            onClick={(e) => e.stopPropagation()}
          >
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => setShowLogDialog(build.id)}
                  />
                }
              >
                <FileClock />
              </TooltipTrigger>
              <TooltipContent>Build logs</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
  ]

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <FileClock className="h-4 w-4" />
            Build & Deploy History
          </CardTitle>
          <CardDescription>View and manage builds for this application</CardDescription>
        </CardHeader>
        <CardContent>
          {!isLoading && (!builds || builds.length === 0) ? (
            <EmptyState
              title="No builds yet"
              description={repoId
                ? "Click \"Build\" to trigger the first build"
                : "Configure build settings above to get started"}
              icon={Package}
            />
          ) : (
            <DataTable
              columns={columns}
              data={builds}
              isLoading={isLoading}
              onRefresh={refetch}
              manualPagination
              pagination={pagination}
              onPaginationChange={setPagination}
              totalCount={totalCount}
            />
          )}
        </CardContent>
      </Card>

      {repoId && projectId && (
        <UnifiedBuildDeployDialog
          open={triggerBuildDialogOpen}
          onOpenChange={setTriggerBuildDialogOpen}
          repoId={repoId}
          projectId={projectId}
          preSelectedBuildId={selectedBuildId}
          preSelectedDeployEnvId={app?.env_id}
          preSelectedDeployAppId={appId}
        />
      )}

      {/* Build Log Dialog */}
      {repoId && showLogDialog && (
        <Dialog open={!!showLogDialog} onOpenChange={() => setShowLogDialog(null)}>
          <DialogContent className="sm:max-w-[90vw] w-full sm:max-h-[90vh] flex flex-col">
            <DialogHeader>
              <DialogTitle>Build logs</DialogTitle>
            </DialogHeader>
            <BuildLogViewer buildId={showLogDialog} repoId={repoId} />
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}
