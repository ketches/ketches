import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import { Bot, Clock, Loader2, Plus } from "lucide-react"
import * as React from "react"
import { useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"

import { builderSessionsApi, builderRunStatusColors, builderRunStatusLabels, type BuilderSession, type BuilderRunStatus } from "@/api/builder-sessions"
import { envsApi, type Env } from "@/api/envs"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

function RunStatusBadge({ status }: { status: string }) {
  const color = builderRunStatusColors[status as BuilderRunStatus] || "bg-gray-100 text-gray-800"
  const label = builderRunStatusLabels[status as BuilderRunStatus] || status
  return <Badge className={`${color} text-xs font-medium`}>{label}</Badge>
}

export function BuilderSessionsPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [createOpen, setCreateOpen] = React.useState(false)
  const [formData, setFormData] = React.useState({ title: "", prompt: "", buildEnvId: "" })
  const [formErrors, setFormErrors] = React.useState<Record<string, string>>({})

  const { data: sessionsResponse, refetch, isLoading } = useQuery({
    queryKey: ["builder-sessions", projectId],
    queryFn: () => builderSessionsApi.list(projectId!),
    enabled: !!projectId,
  })
  const sessions = sessionsResponse?.items ?? []

  const createMutation = useMutation({
    mutationFn: () =>
      builderSessionsApi.create(projectId!, {
        build_env_id: formData.buildEnvId,
        title: formData.title || undefined,
        prompt: formData.prompt,
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["builder-sessions", projectId] })
      setCreateOpen(false)
      setFormData({ title: "", prompt: "", buildEnvId: "" })
      setFormErrors({})
      toast.success("Session created")
      navigate(`/projects/${projectId}/builder-sessions/${data.session.id}`)
    },
    onError: (error: unknown) => {
      const msg = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to create session", { description: msg })
    },
  })

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {}
    if (!formData.buildEnvId) errors.buildEnvId = "Build environment is required"
    if (!formData.prompt.trim()) errors.prompt = "Prompt is required"
    setFormErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleCreate = () => {
    if (!validateForm()) return
    createMutation.mutate()
  }

  const columns = [
    {
      accessorKey: "title",
      header: "Title",
      cell: ({ row }: { row: { original: BuilderSession } }) => (
        <div className="flex items-center gap-3">
          <Avatar className="h-8 w-8 rounded-lg bg-primary/10 text-primary border-none shrink-0">
            <AvatarFallback className="rounded-lg text-xs font-bold">
              {(row.original.title || "B").charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col min-w-0">
            <span
              className="font-medium text-foreground cursor-pointer hover:text-primary transition-colors truncate"
              onClick={() => navigate(`/projects/${projectId}/builder-sessions/${row.original.id}`)}
            >
              {row.original.title || row.original.id.slice(0, 8)}
            </span>
            {row.original.summary && (
              <span className="text-xs text-muted-foreground truncate max-w-xs">{row.original.summary}</span>
            )}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "latest_run_status",
      header: "Status",
      cell: ({ row }: { row: { original: BuilderSession } }) =>
        row.original.latest_run_status ? (
          <RunStatusBadge status={row.original.latest_run_status} />
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        ),
    },
    {
      accessorKey: "last_activity_at",
      header: "Last Activity",
      cell: ({ row }: { row: { original: BuilderSession } }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3 shrink-0" />
          <span className="text-sm">{formatDate(row.original.last_activity_at)}</span>
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }: { row: { original: BuilderSession } }) => (
        <span className="text-muted-foreground text-sm">{formatDate(row.original.created_at)}</span>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }: { row: { original: BuilderSession } }) => (
        <div className="flex justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate(`/projects/${projectId}/builder-sessions/${row.original.id}`)}
          >
            Open
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={[{ label: "Projects", href: "/projects" }, { label: "Builder Sessions" }]} />

      <div>
        <h1 className="text-2xl font-bold">Builder Sessions</h1>
        <p className="text-sm text-muted-foreground mt-1">
          AI-powered app generation sessions.
        </p>
      </div>

      {isLoading ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : sessions.length === 0 ? (
        <EmptyState
          title="No builder sessions yet"
          description="Start a new session and describe the app you want to build."
          icon={Bot}
          actionText="New Session"
          onAction={() => setCreateOpen(true)}
        />
      ) : (
        <DataTable
          columns={columns}
          data={sessions}
          isLoading={isLoading}
          onRefresh={refetch}
          rightToolbar={() => (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              New Session
            </Button>
          )}
        />
      )}

      <CreateSessionDialog
        open={createOpen}
        onOpenChange={(open) => {
          setCreateOpen(open)
          if (!open) {
            setFormData({ title: "", prompt: "", buildEnvId: "" })
            setFormErrors({})
          }
        }}
        formData={formData}
        formErrors={formErrors}
        onFormDataChange={setFormData}
        onValidate={validateForm}
        onSubmit={handleCreate}
        isSubmitting={createMutation.isPending}
      />
    </div>
  )
}

interface CreateSessionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  formData: { title: string; prompt: string; buildEnvId: string }
  formErrors: Record<string, string>
  onFormDataChange: React.Dispatch<React.SetStateAction<{ title: string; prompt: string; buildEnvId: string }>>
  onValidate: () => boolean
  onSubmit: () => void
  isSubmitting: boolean
}

function CreateSessionDialog({
  open,
  onOpenChange,
  formData,
  formErrors,
  onFormDataChange,
  onSubmit,
  isSubmitting,
}: CreateSessionDialogProps) {
  const { projectId } = useParams<{ projectId: string }>()

  const { data: buildEnvs } = useQuery({
    queryKey: ["builder-build-envs", projectId],
    queryFn: async () => {
      const resp = await envsApi.list(projectId!, { page: 1, page_size: 100 })
      return (resp?.items ?? []).filter((e: Env) => e.is_build_env)
    },
    enabled: !!projectId && open,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>New Builder Session</DialogTitle>
          <DialogDescription>
            Describe the application you want to build and the AI agent will generate it.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="buildEnv">
              Build Environment <span className="text-destructive">*</span>
            </Label>
            <Combobox
              value={formData.buildEnvId}
              onValueChange={(v) => {
                if (v) onFormDataChange((f) => ({ ...f, buildEnvId: v }))
              }}
              itemToStringLabel={(v: string) =>
                buildEnvs?.find((e: Env) => e.id === v)?.name ?? v ?? ""
              }
            >
              <ComboboxInput
                id="buildEnv"
                placeholder="Select a build environment..."
                className="w-full"
              />
              <ComboboxContent>
                <ComboboxList>
                  {!buildEnvs || buildEnvs.length === 0 ? (
                    <ComboboxItem value="__none__" disabled>
                      No build environments available
                    </ComboboxItem>
                  ) : (
                    buildEnvs.map((env: Env) => (
                      <ComboboxItem key={env.id} value={env.id}>
                        {env.name}
                      </ComboboxItem>
                    ))
                  )}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
            {formErrors.buildEnvId && (
              <p className="text-xs text-destructive">{formErrors.buildEnvId}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="bs-title">Title (optional)</Label>
            <Input
              id="bs-title"
              placeholder="My new app"
              value={formData.title}
              onChange={(e) => onFormDataChange((f) => ({ ...f, title: e.target.value }))}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="bs-prompt">
              What do you want to build? <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="bs-prompt"
              placeholder="Describe your app idea... e.g. A todo list app with user authentication and a REST API"
              className="min-h-32"
              value={formData.prompt}
              onChange={(e) => onFormDataChange((f) => ({ ...f, prompt: e.target.value }))}
            />
            {formErrors.prompt && (
              <p className="text-xs text-destructive">{formErrors.prompt}</p>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={isSubmitting}>
            {isSubmitting ? "Creating..." : "Create Session"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
