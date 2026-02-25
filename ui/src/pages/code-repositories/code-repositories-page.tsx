import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  FolderGit2,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  Trash2
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { codeRepositoriesApi, type CodeRepository } from "@/api/code-repositories"
import { CreateCodeRepositoryDialog } from "@/components/code-repositories/create-code-repository-dialog"
import { EditCodeRepositoryDialog } from "@/components/code-repositories/edit-code-repository-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyCodeRepositoryState, EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectStore } from "@/stores/project"

const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

const CODE_REPOS_VIEW_MODE_KEY = "code_repositories_view_mode"

export function CodeRepositoriesPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()
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

  const { data: reposResponse, isLoading, refetch } = useQuery({
    queryKey: ["code-repositories", activeProjectId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => codeRepositoriesApi.list(activeProjectId!, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize
    }),
    enabled: !!activeProjectId,
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
        <div
          className="flex flex-col cursor-pointer group/name"
          onClick={() => navigate(`/code-repositories/${row.original.id}`)}
        >
          <span className="font-medium text-foreground group-hover/name:text-primary transition-colors">
            {row.original.name}
          </span>
          <span className="text-xs text-muted-foreground font-mono truncate max-w-[280px]">
            {row.original.git_repo_url}
          </span>
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => (
        <span className="text-muted-foreground">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end">
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={(e) => {
                  e.stopPropagation()
                  setEditingRepo(row.original)
                  setEditDialogOpen(true)
                }}
              >
                <Pencil />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Edit</p>
            </TooltipContent>
          </Tooltip>
        </div>
      ),
    },
  ]

  const breadcrumbs = [{ label: "Code Repositories", icon: FolderGit2 }]

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <PageHeader items={breadcrumbs} />
        <EmptyState
          title="Select a project"
          description="Select a project to view and manage code repositories."
          icon={FolderGit2}
        />
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6 animate-pulse">
        <div className="flex flex-col gap-2">
          <div className="h-8 w-48 bg-muted rounded" />
          <div className="h-4 w-64 bg-muted rounded" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-48 bg-muted rounded-lg" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {!isLoading && safeRepos.length === 0 ? (
        <EmptyCodeRepositoryState onAction={() => setCreateOpen(true)} />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Code Repositories</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage Git repositories, build configs, and deployments
              </p>
            </div>
          </div>

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
                <Tabs value={viewMode} onValueChange={(v) => setViewMode(v as "list" | "card")} className="w-auto h-7">
                  <TabsList>
                    <TabsTrigger value="list">
                      <ListIcon />
                    </TabsTrigger>
                    <TabsTrigger value="card">
                      <LayoutGrid />
                    </TabsTrigger>
                  </TabsList>
                </Tabs>
                <Button onClick={() => setCreateOpen(true)}>
                  <Plus />
                  Add Repository
                </Button>
              </div>
            )}
            renderCard={(repo) => (
              <Card
                key={repo.id}
                className="group/card hover:shadow-md transition-shadow cursor-pointer h-full"
                onClick={() => navigate(`/code-repositories/${repo.id}`)}
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
                          <CardTitle className="text-base font-semibold truncate">
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
                          {repo.description && (
                            <>
                              <span>•</span>
                              <span className="truncate">{repo.description}</span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 group-hover/card:opacity-100 transition-opacity shrink-0"
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingRepo(repo)
                          setEditDialogOpen(true)
                        }}
                      >
                        <Pencil />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={(e) => {
                          e.stopPropagation()
                          setDeletingRepo(repo)
                          setDeleteDialogOpen(true)
                        }}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-2">
                  <div className="text-[10px] text-muted-foreground truncate font-mono mb-2">
                    {repo.git_repo_url}
                  </div>
                  <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                    <span>Created at {formatDate(repo.created_at)}</span>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
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
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingRepo) {
                  deleteMutation.mutate(deletingRepo.id)
                }
                setDeleteDialogOpen(false)
                setDeletingRepo(null)
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
