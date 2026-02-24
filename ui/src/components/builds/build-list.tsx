import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
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
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface BuildListProps {
  appId: string
}

export function BuildList({ appId }: BuildListProps) {
  const queryClient = useQueryClient()
  const [showLogDialog, setShowLogDialog] = React.useState<string | null>(null)
  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildId, setSelectedBuildId] = React.useState<string | undefined>(undefined)
  const [currentPage, setCurrentPage] = React.useState(1)
  const itemsPerPage = 10

  const { data: app } = useQuery({
    queryKey: ['app', appId],
    queryFn: () => appsApi.get(appId),
  })

  const { data: builds, isLoading } = useQuery({
    queryKey: ['builds', appId],
    queryFn: () => buildsApi.list(appId),
    refetchInterval: 5000,
  })

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
    onError: (err: any) => {
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
            <>
              <div className="border-y border-x-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">#</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Git Ref</TableHead>
                      <TableHead>Image</TableHead>
                      <TableHead>Duration</TableHead>
                      <TableHead>Trigger</TableHead>
                      <TableHead>Time</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(builds || []).slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage).map((build: Build) => (
                      <TableRow
                        key={build.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => setShowLogDialog(build.id)}
                      >
                        <TableCell>
                          <span className="flex items-center gap-1 text-muted-foreground">
                            <Hash className="h-3 w-3" />{build.build_number}
                          </span>
                        </TableCell>
                        <TableCell>
                          <BuildStatusBadge status={build.status} />
                        </TableCell>
                        <TableCell>
                          <span className="flex items-center gap-1 text-sm">
                            <FolderGit2 className="h-3 w-3" />{build.git_ref}
                          </span>
                        </TableCell>
                        <TableCell>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger>
                                <span className="text-xs font-mono truncate max-w-48 block">
                                  {build.image_full_name || '-'}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>{build.image_full_name}</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </TableCell>
                        <TableCell>
                          {build.duration > 0 ? (
                            <span className="flex items-center gap-1 text-sm">
                              <Clock className="h-3 w-3" />{formatDuration(build.duration)}
                            </span>
                          ) : build.status === 'building' || build.status === 'cloning' ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : '-'}
                        </TableCell>
                        <TableCell>
                          <span className="flex items-center gap-1 text-sm capitalize">
                            <User className="h-3 w-3" />{build.trigger_type}
                          </span>
                        </TableCell>
                        <TableCell>
                          <span className="text-xs text-muted-foreground">
                            {new Date(build.created_at).toLocaleString()}
                          </span>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center gap-1 justify-end" onClick={(e) => e.stopPropagation()}>
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
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {builds.length > itemsPerPage && (
                <Pagination>
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                        className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                      />
                    </PaginationItem>
                    {Array.from({ length: Math.ceil(builds.length / itemsPerPage) }, (_, i) => i + 1).map((page) => (
                      <PaginationItem key={page}>
                        <PaginationLink
                          onClick={() => setCurrentPage(page)}
                          isActive={currentPage === page}
                          className="cursor-pointer"
                        >
                          {page}
                        </PaginationLink>
                      </PaginationItem>
                    ))}
                    <PaginationItem>
                      <PaginationNext
                        onClick={() => setCurrentPage(Math.min(Math.ceil(builds.length / itemsPerPage), currentPage + 1))}
                        className={currentPage >= Math.ceil(builds.length / itemsPerPage) ? "pointer-events-none opacity-50" : "cursor-pointer"}
                      />
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              )}
            </>
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

