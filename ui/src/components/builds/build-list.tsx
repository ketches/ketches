import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, FileClock, FolderGit2, Hash, Loader2, Package, Rocket, RotateCcw, Square, User } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi } from "@/api/apps"
import { buildConfigsApi } from "@/api/build-configs"
import { buildsApi, type Build } from "@/api/builds"
import { BuildLogViewer } from "@/components/builds/build-log-viewer"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { EmptyState } from "@/components/shared/empty-state"
import { DataTable } from "@/components/data-table/data-table"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import type { AxiosError } from "axios"

interface BuildListProps {
  appId: string
}

export function BuildList({ appId }: BuildListProps) {
  const queryClient = useQueryClient()
  const [showLogDialog, setShowLogDialog] = React.useState<string | null>(null)
  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildId, setSelectedBuildId] = React.useState<string | undefined>(undefined)
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

  const { data: config } = useQuery({
    queryKey: ['build-config', appId],
    queryFn: () => buildConfigsApi.get(appId),
    retry: false,
  })

  const cancelMutation = useMutation({
    mutationFn: (buildId: string) => buildsApi.cancel(appId, buildId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['builds', appId] })
      toast.success('Build cancelled')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to cancel build')
    },
  })

  const hasConfig = !!config
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
          <Hash className="h-3 w-3" />{row.original.build_number}
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
        <span className="flex items-center gap-1 text-sm">
          <FolderGit2 className="h-3 w-3" />{row.original.git_ref}
        </span>
      ),
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger>
              <span className="text-xs font-mono truncate max-w-48 block">
                {row.original.image_full_name || '-'}
              </span>
            </TooltipTrigger>
            <TooltipContent>{row.original.image_full_name}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ),
    },
    {
      accessorKey: "duration",
      header: "Duration",
      cell: ({ row }) => {
        const build = row.original
        if (build.duration > 0) {
          return (
            <span className="flex items-center gap-1 text-sm">
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
        <span className="flex items-center gap-1 text-sm capitalize">
          <User className="h-3 w-3" />{row.original.trigger_type}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {new Date(row.original.created_at).toLocaleString()}
        </span>
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
            {(build.status === 'pending' || build.status === 'cloning' || build.status === 'building') && (
              <Button variant="ghost" size="icon-sm" onClick={() => cancelMutation.mutate(build.id)} title="Cancel">
                <Square />
              </Button>
            )}
            {build.status === 'succeeded' && (
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => {
                  setSelectedBuildId(build.id)
                  setTriggerBuildDialogOpen(true)
                }}
                title="Deploy"
              >
                <Rocket />
              </Button>
            )}
            {(build.status === 'failed' || build.status === 'succeeded') && (
              <Button variant="ghost" size="icon-sm" onClick={() => {
                buildsApi.rebuild(appId, build.id).then(() => {
                  queryClient.invalidateQueries({ queryKey: ['builds', appId] })
                  toast.success('Rebuild triggered')
                }).catch((err) => {
                  toast.error(err?.response?.data?.error || 'Failed to rebuild')
                })
              }} title="Rebuild">
                <RotateCcw />
              </Button>
            )}
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
            Deploy History
          </CardTitle>
          <CardDescription>View and manage builds for this application</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : !builds || builds.length === 0 ? (
            <EmptyState
              title="No builds yet"
              description={hasConfig || repoId
                ? "Click \"Build\" to trigger the first build"
                : "Configure build settings above to get started"}
              icon={Package}
            />
          ) : (
            <DataTable
              columns={columns}
              data={builds}
              borderless
              onRowClick={(build) => setShowLogDialog(build.id)}
              getRowClassName={() => "cursor-pointer hover:bg-muted/50"}
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
      {showLogDialog && (
        <Dialog open={!!showLogDialog} onOpenChange={() => setShowLogDialog(null)}>
          <DialogContent className="max-w-4xl max-h-[80vh]">
            <DialogHeader>
              <DialogTitle>Build Logs</DialogTitle>
            </DialogHeader>
            <BuildLogViewer appId={appId} buildId={showLogDialog} />
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}
