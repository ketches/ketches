import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  Clock,
  Copy,
  FolderGit2,
  LayoutGrid,
  Link,
  List as ListIcon,
  Loader2,
  Pencil,
  Plus,
  Trash2
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { codeRepositoriesApi, type CodeRepository } from "@/api/code-repositories"
import { projectsApi } from "@/api/projects"
import { CreateCodeRepositoryDialog } from "@/components/code-repositories/create-code-repository-dialog"
import { EditCodeRepositoryDialog } from "@/components/code-repositories/edit-code-repository-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useProjectStore } from "@/stores/project"

const CODE_REPOS_VIEW_MODE_KEY = "code_repositories_view_mode"

export function CodeRepositoriesPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { activeProjectId: activeProjectIdFromStore } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  // Fetch project info when embedded in a project detail tab (for breadcrumb state)
  const { data: project } = useQuery({
    queryKey: ["project", projectIdProp],
    queryFn: () => projectsApi.get(projectIdProp!),
    enabled: !!projectIdProp,
  })
  const [createOpen, setCreateOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingRepo, setEditingRepo] = React.useState<CodeRepository | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingRepo, setDeletingRepo] = React.useState<CodeRepository | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(CODE_REPOS_VIEW_MODE_KEY)
    return saved === "list" || saved === "card" ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(CODE_REPOS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: reposResponse, refetch, isLoading } = useQuery({
    queryKey: ["code-repositories", activeProjectId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => codeRepositoriesApi.list(activeProjectId!, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize
    }),
    enabled: !!activeProjectId,
    placeholderData: (previousData) => previousData,
  })

  const repos = reposResponse?.items ?? []
  const paginationInfo = reposResponse?.pagination

  const deleteMutation = useMutation({
    mutationFn: (id: string) => codeRepositoriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["code-repositories", activeProjectId] })
      toast.success("Code repository deleted")
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data
            ?.error
          : null
      toast.error(msg || "Failed to delete code repository")
    },
  })

  const safeRepos = Array.isArray(repos) ? repos : []

  const columns: ColumnDef<CodeRepository>[] = [
    {
      accessorKey: "name",
      header: "Repository",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-lime-500/10 rounded-md text-lime-600 shrink-0">
            <FolderGit2 className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <span className="font-medium text-foreground cursor-pointer hover:text-primary transition-colors"
              onClick={() => navigate(`/code-repositories/${row.original.id}`, { state: projectIdProp && project ? { fromProjectId: projectIdProp, fromProjectName: project.name } : undefined })}>{row.original.name}</span>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.slug}
            </p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "git_repo_url",
      header: "Git URL",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground font-mono truncate max-w-100">
            {row.original.git_repo_url}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/card:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(row.original.git_repo_url)
              toast.success("Git repository URL copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
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
  ]

  if (!isViewer) {
    columns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setEditingRepo(row.original)
                    setEditDialogOpen(true)
                  }}
                />
              }
            >
              <Pencil />
            </TooltipTrigger>
            <TooltipContent>Edit repository</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeletingRepo(row.original)
                    setDeleteDialogOpen(true)
                  }}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Delete repository</TooltipContent>
          </Tooltip>
        </div>
      ),
    })
  }

  const breadcrumbs = [{ label: "Code Repositories", icon: FolderGit2 }]

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        {!projectIdProp && <PageHeader items={breadcrumbs} />}
        <EmptyState
          title="Select a project"
          description="Select a project to view and manage code repositories."
          icon={FolderGit2}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      {!projectIdProp && <PageHeader items={breadcrumbs} />}

      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Code Repositories</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage Git repositories, build settings, and deployments
            </p>
          </div>
        </div>
      )}

      {isLoading && !reposResponse ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={safeRepos}
          viewMode={viewMode}
          onRefresh={refetch}
          manualPagination
          totalCount={paginationInfo?.total || 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          leftActions={() => (
            <Input
              className="flex flex-1 max-w-sm min-w-75"
              placeholder="Search repositories..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          )}
          toolbarActions={() => (
            <div className="flex items-center gap-2">
              <Tabs value={viewMode} onValueChange={(v) => {
                const newMode = v as "list" | "card"
                setViewMode(newMode)
                setPagination((prev) => ({
                  ...prev,
                  pageIndex: 0,
                  pageSize: newMode === "card" ? 9 : 10,
                }))
              }} className="w-auto h-7">
                <TabsList>
                  <TabsTrigger value="list">
                    <ListIcon />
                  </TabsTrigger>
                  <TabsTrigger value="card">
                    <LayoutGrid />
                  </TabsTrigger>
                </TabsList>
              </Tabs>
              {!isViewer && (
                <Button onClick={() => setCreateOpen(true)}>
                  <Plus />
                  Add Repository
                </Button>
              )}
            </div>
          )}
          renderCard={(repo) => (
            <Card
              key={repo.id}
              className="group/card hover:shadow-md transition-shadow h-full"
            >
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3 min-w-0">
                    <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none shrink-0">
                      <AvatarFallback className="rounded-lg text-lg font-bold">
                        {repo.name.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex flex-col min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <CardTitle className="text-base font-semibold truncate cursor-pointer hover:text-primary transition-colors"
                          onClick={() => navigate(`/code-repositories/${repo.id}`, { state: projectIdProp && project ? { fromProjectId: projectIdProp, fromProjectName: project.name } : undefined })}>
                          {repo.name}
                        </CardTitle>
                        {repo.webhook_enabled && (
                          <span className="text-[10px] text-muted-foreground px-1.5 py-0 rounded-full bg-muted border shrink-0">
                            Webhook enabled
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                        <span>{repo.slug}</span>
                        <span>•</span>
                        {repo.description ? (
                          <span className="truncate">{repo.description}</span>
                        ) : (
                          <span className="italic">No description</span>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                    {!isViewer && (
                      <Tooltip>
                        <TooltipTrigger
                          delay={200}
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={(e) => {
                                e.stopPropagation()
                                setEditingRepo(repo)
                                setEditDialogOpen(true)
                              }}
                            />
                          }
                        >
                          <Pencil />
                        </TooltipTrigger>
                        <TooltipContent>Edit repository</TooltipContent>
                      </Tooltip>
                    )}
                    {!isViewer && (
                      <Tooltip>
                        <TooltipTrigger
                          delay={200}
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              className="text-destructive hover:text-destructive hover:bg-destructive/10"
                              onClick={(e) => {
                                e.stopPropagation()
                                setDeletingRepo(repo)
                                setDeleteDialogOpen(true)
                              }}
                            />
                          }
                        >
                          <Trash2 />
                        </TooltipTrigger>
                        <TooltipContent>Delete repository</TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4 pt-2">
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Link className="h-3.5 w-3.5" />
                    <span className="font-mono">
                      {repo.git_repo_url}
                    </span>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigator.clipboard.writeText(repo.git_repo_url)
                        toast.success("Git repository URL copied to clipboard")
                      }}
                    >
                      <Copy />
                    </Button>
                  </div>
                </div>

                <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                  <div className="flex items-center gap-1.5">
                    <Clock className="h-3 w-3" />
                    <span>Created at {formatDate(repo.created_at)}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        />
      )}

      <CreateCodeRepositoryDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={activeProjectId}
        onSuccess={(repo) => navigate(`/code-repositories/${repo.id}`)}
      />
      <EditCodeRepositoryDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        repo={editingRepo}
        onSuccess={() => {
          setEditingRepo(null)
        }}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Code Repository</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingRepo
                ? `Delete code repository "${deletingRepo.name}"? This action cannot be undone.`
                : "Are you sure you want to delete this code repository?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingRepo) {
                  deleteMutation.mutate(deletingRepo.id)
                }
                setDeleteDialogOpen(false)
                setDeletingRepo(null)
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
