import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import {
  FileText,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { templatesApi, type Template } from "@/api/templates"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { CreateTemplateDialog } from "@/components/templates/create-template-dialog"
import { EditTemplateDialog } from "@/components/templates/edit-template-dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
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

const statusConfig: Record<
  string,
  { label: string; color: "blue" | "yellow" | "green" | "red" | "gray" }
> = {
  draft: { label: "Draft", color: "gray" },
  reviewing: { label: "Reviewing", color: "yellow" },
  published: { label: "Published", color: "green" },
  deprecated: { label: "Deprecated", color: "red" },
}

const typeConfig: Record<string, { label: string; color: "blue" | "purple" | "orange" | "sky" }> = {
  application: { label: "Application", color: "blue" },
  service: { label: "Service", color: "purple" },
  job: { label: "Job", color: "orange" },
  cronjob: { label: "CronJob", color: "sky" },
}

function TemplateStatusBadge({ status }: { status: string }) {
  const config = statusConfig[status] ?? { label: status, color: "gray" as const }
  return <ColorBadge color={config.color}>{config.label}</ColorBadge>
}

function TemplateTypeBadge({ type }: { type: string }) {
  const config = typeConfig[type] ?? { label: type, color: "blue" as const }
  return <ColorBadge color={config.color}>{config.label}</ColorBadge>
}

const TEMPLATES_VIEW_MODE_KEY = "templates_view_mode"

export function TemplatesPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()
  const [createOpen, setCreateOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingTemplate, setEditingTemplate] = React.useState<Template | null>(
    null
  )
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingTemplate, setDeletingTemplate] =
    React.useState<Template | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(TEMPLATES_VIEW_MODE_KEY)
    return saved === "list" || saved === "card" ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(TEMPLATES_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const { data: templates = [], isLoading, refetch } = useQuery({
    queryKey: ["templates", activeProjectId],
    queryFn: () => templatesApi.list(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => templatesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["templates", activeProjectId] })
      toast.success("Template deleted")
      setDeleteDialogOpen(false)
      setDeletingTemplate(null)
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data
            ?.error
          : null
      toast.error(msg || "Failed to delete template")
    },
  })

  const safeTemplates = Array.isArray(templates) ? templates : []
  const filteredTemplates = React.useMemo(() => {
    if (!debouncedSearch.trim()) return safeTemplates
    const q = debouncedSearch.toLowerCase()
    return safeTemplates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.slug.toLowerCase().includes(q) ||
        t.description?.toLowerCase().includes(q) ||
        t.type.toLowerCase().includes(q)
    )
  }, [safeTemplates, debouncedSearch])

  const columns: ColumnDef<Template>[] = [
    {
      accessorKey: "name",
      header: "Template",
      cell: ({ row }) => (
        <div
          className="flex flex-col cursor-pointer group/name"
          onClick={() => navigate(`/templates/${row.original.id}`)}
        >
          <span className="font-medium text-foreground group-hover/name:text-primary transition-colors">
            {row.original.name}
          </span>
          <span className="text-xs text-muted-foreground font-mono truncate max-w-[280px]">
            {row.original.slug}
          </span>
        </div>
      ),
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <TemplateTypeBadge type={row.original.type} />,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <TemplateStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "enabled",
      header: "Enabled",
      cell: ({ row }) => (
        <span
          className={
            row.original.enabled ? "text-green-600" : "text-muted-foreground"
          }
        >
          {row.original.enabled ? "Yes" : "No"}
        </span>
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
        <div className="flex items-center justify-end gap-1">
          <Tooltip>
            <TooltipTrigger render={<Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setEditingTemplate(row.original); setEditDialogOpen(true) }} />}>
              <Pencil />
            </TooltipTrigger>
            <TooltipContent>
              <p>Edit</p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger render={<Button variant="ghost" size="icon-sm" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={(e) => { e.stopPropagation(); setDeletingTemplate(row.original); setDeleteDialogOpen(true) }} />}>
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>
              <p>Delete</p>
            </TooltipContent>
          </Tooltip>
        </div>
      ),
    },
  ]

  const breadcrumbs = [{ label: "Templates", icon: FileText }]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search templates..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs
        value={viewMode}
        onValueChange={(v) => setViewMode(v as "list" | "card")}
        className="w-auto h-7"
      >
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
        Create Template
      </Button>
    </div>
  )

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <PageHeader items={breadcrumbs} />
        <EmptyState
          title="Select a project"
          description="Select a project to view and manage templates."
          icon={FileText}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {!isLoading && safeTemplates.length === 0 ? (
        <EmptyState
          title="No templates yet"
          description="Create your first template to define reusable configurations for your resources."
          icon={FileText}
          actionText="Create Template"
          onAction={() => setCreateOpen(true)}
          actionIcon={Plus}
        />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Templates</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage reusable templates for applications, services, and other
                resources
              </p>
            </div>
          </div>

          <DataTable
            columns={columns}
            data={filteredTemplates}
            viewMode={viewMode}
            onRefresh={refetch}
            leftActions={() => toolbarLeft}
            toolbarActions={() => toolbarRight}
            renderCard={(template) => (
              <Card
                key={template.id}
                className="group/card hover:shadow-md transition-shadow cursor-pointer h-full"
                onClick={() => navigate(`/templates/${template.id}`)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 min-w-0">
                      <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none shrink-0">
                        <AvatarFallback className="rounded-lg text-lg font-bold">
                          {template.name.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <CardTitle className="text-base font-semibold truncate">
                            {template.name}
                          </CardTitle>
                          <TemplateStatusBadge status={template.status} />
                          {!template.enabled && (
                            <ColorBadge color="gray">Disabled</ColorBadge>
                          )}
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                          <span>{template.slug}</span>
                          {template.description && (
                            <>
                              <span>•</span>
                              <span className="truncate">{template.description}</span>
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
                          setEditingTemplate(template)
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
                          setDeletingTemplate(template)
                          setDeleteDialogOpen(true)
                        }}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-2">
                  <div className="flex items-center gap-2 mb-2">
                    <TemplateTypeBadge type={template.type} />
                  </div>
                  <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                    <span>Created at {formatDate(template.created_at)}</span>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
      )}

      <CreateTemplateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={activeProjectId}
        onSuccess={(template) => navigate(`/templates/${template.id}`)}
      />
      <EditTemplateDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        template={editingTemplate}
        onSuccess={() => {
          setEditingTemplate(null)
        }}
      />

      {/* Delete confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Template</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{" "}
              <strong>{deletingTemplate?.name}</strong>? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => {
                if (deletingTemplate) {
                  deleteMutation.mutate(deletingTemplate.id)
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
