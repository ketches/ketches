import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ChevronsUpDown,
  Code,
  Copy,
  FileText,
  Info,
  Pencil,
  Telescope,
  Trash2,
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { templatesApi } from "@/api/templates"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
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
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
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

const typeConfig: Record<
  string,
  { label: string; color: "blue" | "purple" | "orange" | "sky" }
> = {
  application: { label: "Application", color: "blue" },
  service: { label: "Service", color: "purple" },
  job: { label: "Job", color: "orange" },
  cronjob: { label: "CronJob", color: "sky" },
}

function TemplateStatusBadge({ status }: { status: string }) {
  const config = statusConfig[status] ?? {
    label: status,
    color: "gray" as const,
  }
  return <ColorBadge color={config.color}>{config.label}</ColorBadge>
}

function TemplateTypeBadge({ type }: { type: string }) {
  const config = typeConfig[type] ?? { label: type, color: "blue" as const }
  return <ColorBadge color={config.color}>{config.label}</ColorBadge>
}

export function TemplateDetailPage() {
  const { templateId } = useParams<{ templateId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()

  const currentTab = searchParams.get("tab") || "overview"
  const { activeProjectId } = useProjectStore()
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  const { data: template, isLoading } = useQuery({
    queryKey: ["template", templateId],
    queryFn: () => templatesApi.get(templateId!),
    enabled: !!templateId,
  })

  const { data: templates = [] } = useQuery({
    queryKey: ["templates", activeProjectId],
    queryFn: () => templatesApi.list(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const safeTemplates = Array.isArray(templates) ? templates : []

  const deleteMutation = useMutation({
    mutationFn: () => templatesApi.delete(templateId!),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["templates", activeProjectId],
      })
      toast.success("Template deleted")
      navigate("/templates")
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

  if (!templateId) {
    return (
      <NotFoundPage
        resourceType="Template"
        backHref="/templates"
        backLabel="Back to Templates"
      />
    )
  }

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6 animate-pulse">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-32" />
          <div className="flex justify-between items-start">
            <div className="flex items-center gap-4">
              <Skeleton className="h-14 w-14 rounded-lg" />
              <div className="space-y-2">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-4 w-64" />
              </div>
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-full max-w-50" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (!template) {
    return (
      <NotFoundPage
        resourceType="Template"
        backHref="/templates"
        backLabel="Back to Templates"
      />
    )
  }

  const breadcrumbs = [
    {
      label: "Templates",
      href: "/templates",
      icon: FileText,
    },
    {
      label: template.name,
      icon: FileText,
      dropdown:
        safeTemplates.length > 1 ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon-sm">
                  <ChevronsUpDown />
                </Button>
              }
            />
            <DropdownMenuContent align="start" className="w-48">
              <DropdownMenuGroup>
                {safeTemplates.map((t) => (
                  <DropdownMenuItem
                    key={t.id}
                    onClick={() => navigate(`/templates/${t.id}`)}
                  >
                    <FileText className="mr-2 h-4 w-4" />
                    {t.name}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : undefined,
    },
  ]

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {/* Header */}
      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary shrink-0">
              <FileText className="h-8 w-8" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight truncate">
                  {template.name}
                </h1>

                <TemplateTypeBadge type={template.type} />
                <TemplateStatusBadge status={template.status} />
                {!template.enabled && <ColorBadge color="gray">Disabled</ColorBadge>}
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="font-mono">{template.slug}</span>
                {template.description && (
                  <>
                    <span>•</span>
                    <span>{template.description}</span>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setEditDialogOpen(true)}
            >
              <Pencil />
              Edit
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setDeleteDialogOpen(true)}
            >
              <Trash2 />
              Delete
            </Button>
          </div>
        </div>
      </div>

      <Tabs value={currentTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview"><Telescope />Overview</TabsTrigger>
          <TabsTrigger value="content"><Code />Content</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-primary/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Info className="h-4 w-4" />
                Template Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Slug</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono">{template.slug}</p>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => {
                        navigator.clipboard.writeText(template.slug)
                        toast.success("Copied to clipboard")
                      }}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Type</p>
                  <p className="text-sm">{template.type}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Created At</p>
                  <p className="text-sm">{template.created_at ? formatDate(template.created_at) : "N/A"}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Content Tab */}
        <TabsContent value="content" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Code className="h-4 w-4" />
                Template Content
              </CardTitle>
              <CardDescription>
                The raw content of this template (YAML, JSON, or other format).
              </CardDescription>
            </CardHeader>
            <CardContent>
              {template.content ? (
                <pre className="bg-muted rounded-lg p-4 overflow-auto text-sm font-mono whitespace-pre-wrap max-h-150">
                  <code>{template.content}</code>
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground italic">
                  No content defined yet.
                </p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <EditTemplateDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        template={template}
        onSuccess={() =>
          queryClient.invalidateQueries({
            queryKey: ["template", templateId],
          })
        }
      />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Template</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{" "}
              <strong>{template.name}</strong>? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => deleteMutation.mutate()}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div >
  )
}
