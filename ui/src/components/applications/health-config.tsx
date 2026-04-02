import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Edit2, HeartPulse, Loader2, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App, ProbeSpec } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
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
import { getErrorMessage } from "@/lib/utils"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useProjectRole } from "@/hooks/useProjectRole"
import { ColorBadge } from "../shared/color-badge"

const PROBE_TYPE_OPTIONS = [
  { value: "readiness", label: "READINESS" },
  { value: "liveness", label: "LIVENESS" },
  { value: "startup", label: "STARTUP" },
] as const

const PROBE_MODE_OPTIONS = [
  { value: "httpGet", label: "HTTP GET" },
  { value: "tcpSocket", label: "TCP Socket" },
  { value: "exec", label: "Exec Command" },
] as const

const DEFAULT_PROBE: Omit<ProbeSpec, "type"> = {
  probe_mode: "httpGet",
  enabled: true,
  http_get_path: "/",
  http_get_port: 80,
  tcp_socket_port: 80,
  exec_command: "",
  initial_delay_seconds: 0,
  period_seconds: 10,
  timeout_seconds: 1,
  success_threshold: 1,
  failure_threshold: 3,
}

interface ProbeEditorDialogProps {
  app: App
  /** The probe being edited; null means "add new" */
  probe: ProbeSpec | null
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Probe types already in use (excluding the one being edited) */
  usedTypes: ProbeSpec["type"][]
  onSuccess: () => void
}

function ProbeEditorDialog({
  app,
  probe,
  open,
  onOpenChange,
  usedTypes,
  onSuccess,
}: ProbeEditorDialogProps) {
  const queryClient = useQueryClient()
  const isEditing = probe !== null

  // Determine available type options (exclude already-used types except for the current one)
  const availableTypeOptions = PROBE_TYPE_OPTIONS.filter(
    (opt) => !usedTypes.includes(opt.value as ProbeSpec["type"])
  )

  const [formData, setFormData] = React.useState<ProbeSpec>({
    type: "readiness",
    ...DEFAULT_PROBE,
  })

  // Reset form when dialog opens
  React.useEffect(() => {
    if (open) {
      if (isEditing && probe) {
        setFormData({ ...probe })
      } else {
        // Pick the first available type that hasn't been used yet
        const firstAvailable = availableTypeOptions[0]?.value as ProbeSpec["type"] ?? "readiness"
        setFormData({
          type: firstAvailable,
          ...DEFAULT_PROBE,
        })
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isEditing])

  const saveMutation = useMutation({
    mutationFn: async (data: ProbeSpec) => {
      // Build the new probes list
      const currentProbes: ProbeSpec[] = app.probes ?? []
      let updatedProbes: ProbeSpec[]

      if (isEditing && probe) {
        // Replace the probe with matching type
        updatedProbes = currentProbes.map((p) =>
          p.type === probe.type ? data : p
        )
      } else {
        // Add the new probe
        updatedProbes = [...currentProbes, data]
      }

      return appsApi.updateHealth(app.id, updatedProbes)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app", app.id] })
      toast.success(isEditing ? "Probe updated successfully" : "Probe created successfully")
      onOpenChange(false)
      onSuccess()
    },
    onError: (error: unknown) => {
      toast.error("Failed to save probe", {
        description: getErrorMessage(error, "Unknown error"),
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    // Clean up fields not relevant to the selected mode
    const cleaned = { ...formData }
    if (cleaned.probe_mode !== "httpGet") {
      cleaned.http_get_path = undefined
      cleaned.http_get_port = undefined
    }
    if (cleaned.probe_mode !== "tcpSocket") {
      cleaned.tcp_socket_port = undefined
    }
    if (cleaned.probe_mode !== "exec") {
      cleaned.exec_command = undefined
    }
    saveMutation.mutate(cleaned)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? "Edit Probe" : "Add Probe"}</DialogTitle>
            <DialogDescription>
              Configure health check probe for your application container.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Type + Enabled */}
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Probe Type *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.type}
                    onValueChange={(v) =>
                      v && setFormData((prev) => ({ ...prev, type: v as ProbeSpec["type"] }))
                    }
                    itemToStringLabel={(v) =>
                      PROBE_TYPE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""
                    }
                    disabled={isEditing}
                  >
                    <ComboboxInput />
                    <ComboboxContent>
                      <ComboboxList>
                        {(isEditing ? PROBE_TYPE_OPTIONS : availableTypeOptions).map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            {option.label}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel>Mode *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.probe_mode}
                    onValueChange={(v) =>
                      v && setFormData((prev) => ({ ...prev, probe_mode: v as ProbeSpec["probe_mode"] }))
                    }
                    itemToStringLabel={(v) =>
                      PROBE_MODE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""
                    }
                  >
                    <ComboboxInput />
                    <ComboboxContent>
                      <ComboboxList>
                        {PROBE_MODE_OPTIONS.map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            {option.label}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>
            </div>

            {/* Mode-specific fields */}
            {formData.probe_mode === "httpGet" && (
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>HTTP Path</FieldLabel>
                  <FieldContent>
                    <Input
                      placeholder="/"
                      value={formData.http_get_path ?? ""}
                      onChange={(e) =>
                        setFormData((prev) => ({ ...prev, http_get_path: e.target.value }))
                      }
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>HTTP Port</FieldLabel>
                  <FieldContent>
                    <Input
                      type="number"
                      placeholder="80"
                      value={formData.http_get_port ?? ""}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          http_get_port: parseInt(e.target.value) || undefined,
                        }))
                      }
                    />
                  </FieldContent>
                </Field>
              </div>
            )}

            {formData.probe_mode === "tcpSocket" && (
              <Field>
                <FieldLabel>TCP Port</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    placeholder="80"
                    value={formData.tcp_socket_port ?? ""}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        tcp_socket_port: parseInt(e.target.value) || undefined,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
            )}

            {formData.probe_mode === "exec" && (
              <Field>
                <FieldLabel>Exec Command</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="cat /tmp/healthy"
                    value={formData.exec_command ?? ""}
                    onChange={(e) =>
                      setFormData((prev) => ({ ...prev, exec_command: e.target.value }))
                    }
                  />
                </FieldContent>
              </Field>
            )}

            {/* Timing parameters */}
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              <Field>
                <FieldLabel className="text-xs">Initial Delay (s)</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    value={formData.initial_delay_seconds}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        initial_delay_seconds: parseInt(e.target.value) || 0,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel className="text-xs">Period (s)</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    value={formData.period_seconds}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        period_seconds: parseInt(e.target.value) || 1,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel className="text-xs">Timeout (s)</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    value={formData.timeout_seconds}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        timeout_seconds: parseInt(e.target.value) || 1,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel className="text-xs">Success Thres.</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    value={formData.success_threshold}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        success_threshold: parseInt(e.target.value) || 1,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel className="text-xs">Failure Thres.</FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    value={formData.failure_threshold}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        failure_threshold: parseInt(e.target.value) || 1,
                      }))
                    }
                  />
                </FieldContent>
              </Field>
            </div>
          </div>

          <DialogFooter className="sm:justify-between">
            <div className="flex items-center gap-2">
              <Checkbox
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: !!checked }))}
              />
              <label htmlFor="enabled" className="cursor-pointer">
                Enabled
              </label>
            </div>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {isEditing ? "Update" : "Create"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface HealthConfigProps {
  app: App
}

export function HealthConfig({ app }: HealthConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === "viewer"

  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingProbe, setEditingProbe] = React.useState<ProbeSpec | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingProbe, setDeletingProbe] = React.useState<ProbeSpec | null>(null)

  const probes: ProbeSpec[] = app.probes ?? []
  const allTypesUsed = probes.length >= 3

  // Probe types currently in use, excluding the one being edited
  const usedTypes = probes
    .map((p) => p.type)
    .filter((t): t is ProbeSpec["type"] =>
      editingProbe ? t !== editingProbe.type : true
    )

  const deleteMutation = useMutation({
    mutationFn: (probeType: ProbeSpec["type"]) => {
      const updatedProbes = probes.filter((p) => p.type !== probeType)
      return appsApi.updateHealth(app.id, updatedProbes)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app", app.id] })
      toast.success("Probe deleted successfully")
      setDeleteDialogOpen(false)
      setDeletingProbe(null)
    },
    onError: (error: unknown) => {
      toast.error("Failed to delete probe", {
        description: getErrorMessage(error, "Unknown error"),
      })
    },
  })

  const handleAdd = () => {
    setEditingProbe(null)
    setIsDialogOpen(true)
  }

  const handleEdit = (probe: ProbeSpec) => {
    setEditingProbe(probe)
    setIsDialogOpen(true)
  }

  const handleDelete = (probe: ProbeSpec) => {
    setDeletingProbe(probe)
    setDeleteDialogOpen(true)
  }

  const handleDialogSuccess = () => {
    setIsDialogOpen(false)
    setEditingProbe(null)
  }

  const probeColumns: ColumnDef<ProbeSpec>[] = [
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => (
        <Badge variant="outline" className="uppercase font-medium font-mono">
          {row.original.type}
        </Badge>
      ),
    },
    {
      accessorKey: "probe_mode",
      header: "Mode",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {PROBE_MODE_OPTIONS.find((o) => o.value === row.original.probe_mode)?.label ??
            row.original.probe_mode}
        </span>
      ),
    },
    {
      id: "endpoint",
      header: "Endpoint / Command",
      cell: ({ row }) => {
        const p = row.original
        if (p.probe_mode === "httpGet") {
          return (
            <span className="font-mono text-xs">
              {p.http_get_path ?? "/"} : {p.http_get_port ?? 80}
            </span>
          )
        }
        if (p.probe_mode === "tcpSocket") {
          return (
            <span className="font-mono text-xs">:{p.tcp_socket_port ?? "-"}</span>
          )
        }
        return (
          <span className="font-mono text-xs truncate max-w-40 block">
            {p.exec_command || "-"}
          </span>
        )
      },
    },
    {
      id: "timing",
      header: "Timing",
      cell: ({ row }) => {
        const p = row.original
        return (
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            delay {p.initial_delay_seconds}s · period {p.period_seconds}s · timeout {p.timeout_seconds}s
          </span>
        )
      },
    },
    {
      id: "thresholds",
      header: "Thresholds",
      cell: ({ row }) => {
        const p = row.original
        return (
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            ✓ {p.success_threshold} · ✗ {p.failure_threshold}
          </span>
        )
      },
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ row }) => (
        <ColorBadge color={row.original.enabled ? "green" : "gray"}>
          {row.original.enabled ? "Enabled" : "Disabled"}
        </ColorBadge>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          {!isViewer && (
            <>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => handleEdit(row.original)}
                    />
                  }
                >
                  <Edit2 />
                </TooltipTrigger>
                <TooltipContent>Edit Probe</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => handleDelete(row.original)}
                      disabled={deleteMutation.isPending}
                    />
                  }
                >
                  <Trash2 />
                </TooltipTrigger>
                <TooltipContent>Delete Probe</TooltipContent>
              </Tooltip>
            </>
          )}
        </div>
      ),
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <HeartPulse className="h-4 w-4" /> Health Checks
        </CardTitle>
        <CardDescription>
          Configure Readiness, Liveness, and Startup probes
        </CardDescription>
        <CardAction>
          {!isViewer && probes.length > 0 && (
            <Button
              onClick={handleAdd}
              disabled={allTypesUsed}
              title={allTypesUsed ? "All probe types have been configured" : undefined}
            >
              <Plus />
              Add Probe
            </Button>
          )}
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-4">
        {probes.length === 0 ? (
          <EmptyState
            title="No probes configured"
            description="Add health check probes to monitor your application's readiness and liveness."
            icon={HeartPulse}
            actionText={!isViewer ? "Add Probe" : undefined}
            onAction={!isViewer ? handleAdd : undefined}
            actionIcon={Plus}
          />
        ) : (
          <DataTable
            columns={probeColumns}
            data={probes}
            hidePagination
          />
        )}
      </CardContent>

      {/* Add / Edit dialog */}
      <ProbeEditorDialog
        app={app}
        probe={editingProbe}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        usedTypes={usedTypes}
        onSuccess={handleDialogSuccess}
      />

      {/* Delete confirm dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Probe</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingProbe
                ? `Are you sure you want to delete the ${deletingProbe.type.toUpperCase()} probe? This action cannot be undone.`
                : "Are you sure you want to delete this probe?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingProbe) {
                  deleteMutation.mutate(deletingProbe.type)
                }
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
