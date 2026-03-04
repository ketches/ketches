import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, Edit2, ExternalLink, GamepadDirectional, Loader2, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type ClusterIntegration, type CreateClusterIntegrationRequest, type IntegrationType } from "@/api/clusters"
import { DataTable } from "@/components/data-table/data-table"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import type { ColumnDef } from "@tanstack/react-table"

const INTEGRATION_TYPES: { value: IntegrationType; label: string; description: string }[] = [
  { value: "prometheus", label: "Prometheus", description: "Metrics and monitoring" },
  { value: "grafana", label: "Grafana", description: "Visualization dashboards" },
  { value: "loki", label: "Loki", description: "Log aggregation" },
  { value: "alertmanager", label: "Alertmanager", description: "Alert management" },
]

interface IntegrationFormData {
  integration_type: IntegrationType
  name: string
  access_mode: 'endpoint' | 'service'
  endpoint: string
  namespace: string
  service_name: string
  service_port: string
  username: string
  password: string
  token: string
  ca_cert: string
  skip_tls_verify: boolean
  enabled: boolean
}

const defaultFormData: IntegrationFormData = {
  integration_type: "prometheus",
  name: "",
  access_mode: "endpoint",
  endpoint: "",
  namespace: "default",
  service_name: "",
  service_port: "80",
  username: "",
  password: "",
  token: "",
  ca_cert: "",
  skip_tls_verify: false,
  enabled: true,
}

interface ClusterIntegrationsConfigProps {
  clusterId: string
}

export function ClusterIntegrationsConfig({ clusterId }: ClusterIntegrationsConfigProps) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [editingIntegration, setEditingIntegration] = React.useState<ClusterIntegration | null>(null)
  const [deletingIntegration, setDeletingIntegration] = React.useState<ClusterIntegration | null>(null)
  const [formData, setFormData] = React.useState<IntegrationFormData>(defaultFormData)

  const { data: namespaces = [] } = useQuery({
    queryKey: ["cluster-namespaces", clusterId],
    queryFn: () => clustersApi.listNamespaces(clusterId),
    enabled: dialogOpen && formData.access_mode === "service",
  })

  const { data: services = [] } = useQuery({
    queryKey: ["cluster-services", clusterId, formData.namespace],
    queryFn: () => clustersApi.listServices(clusterId, formData.namespace),
    enabled: dialogOpen && formData.access_mode === "service" && !!formData.namespace,
  })

  const { data: integrations = [], isLoading } = useQuery({
    queryKey: ["cluster-integrations", clusterId],
    queryFn: () => clustersApi.listIntegrations(clusterId),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateClusterIntegrationRequest) => clustersApi.createIntegration(clusterId, data),
    onSuccess: () => {
      toast.success("Integration added")
      queryClient.invalidateQueries({ queryKey: ["cluster-integrations", clusterId] })
      handleCloseDialog()
    },
    onError: (error: any) => {
      toast.error("Failed to add integration", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => clustersApi.updateIntegration(clusterId, id, data),
    onSuccess: () => {
      toast.success("Integration updated")
      queryClient.invalidateQueries({ queryKey: ["cluster-integrations", clusterId] })
      handleCloseDialog()
    },
    onError: (error: any) => {
      toast.error("Failed to update integration", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => clustersApi.deleteIntegration(clusterId, id),
    onSuccess: () => {
      toast.success("Integration deleted")
      queryClient.invalidateQueries({ queryKey: ["cluster-integrations", clusterId] })
      setDeleteOpen(false)
      setDeletingIntegration(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete integration", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleOpenCreate = () => {
    setEditingIntegration(null)
    setFormData(defaultFormData)
    setDialogOpen(true)
  }

  const handleOpenEdit = (integration: ClusterIntegration) => {
    setEditingIntegration(integration)
    setFormData({
      integration_type: integration.integration_type,
      name: integration.name,
      access_mode: integration.service_name ? "service" : "endpoint",
      endpoint: integration.endpoint || "",
      namespace: integration.namespace || "default",
      service_name: integration.service_name || "",
      service_port: integration.service_port?.toString() || "80",
      username: integration.username || "",
      password: "",
      token: "",
      ca_cert: "",
      skip_tls_verify: integration.skip_tls_verify,
      enabled: integration.enabled,
    })
    setDialogOpen(true)
  }

  const handleCloseDialog = () => {
    setDialogOpen(false)
    setEditingIntegration(null)
    setFormData(defaultFormData)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const payload: any = {
      integration_type: formData.integration_type,
      name: formData.name,
      skip_tls_verify: formData.skip_tls_verify,
      enabled: formData.enabled,
    }

    if (formData.access_mode === "service") {
      payload.namespace = formData.namespace
      payload.service_name = formData.service_name
      payload.service_port = parseInt(formData.service_port) || 80
      payload.endpoint = ""
    } else {
      payload.endpoint = formData.endpoint
      payload.namespace = ""
      payload.service_name = ""
      payload.service_port = 0
    }

    if (formData.username) payload.username = formData.username
    if (formData.password) payload.password = formData.password
    if (formData.token) payload.token = formData.token
    if (formData.ca_cert) payload.ca_cert = formData.ca_cert

    if (editingIntegration) {
      updateMutation.mutate({ id: editingIntegration.id, data: payload })
    } else {
      createMutation.mutate(payload)
    }
  }

  const handleOpenDelete = (integration: ClusterIntegration) => {
    setDeletingIntegration(integration)
    setDeleteOpen(true)
  }

  const columns: ColumnDef<ClusterIntegration>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const integration = row.original
        return (
          <div className="flex items-center gap-2">
            <span className="font-medium">{integration.name}</span>
            <ColorBadge color={integration.enabled ? "green" : "gray"}>
              {integration.enabled ? "Active" : "Disabled"}
            </ColorBadge>
          </div>
        )
      },
    },
    {
      accessorKey: "integration_type",
      header: "Type",
      cell: ({ row }) => {
        const typeInfo = INTEGRATION_TYPES.find((t) => t.value === row.original.integration_type)
        return (
          <div>
            <span className="capitalize text-sm">{typeInfo?.label || row.original.integration_type}</span>
            <p className="text-xs text-muted-foreground">{typeInfo?.description}</p>
          </div>
        )
      },
    },
    {
      accessorKey: "endpoint",
      header: "Endpoint",
      cell: ({ row }) => {
        const integration = row.original
        return (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <ExternalLink className="h-3 w-3 shrink-0" />
            <span className="font-mono truncate max-w-48">
              {integration.service_name
                ? `${integration.namespace}/${integration.service_name}:${integration.service_port}`
                : integration.endpoint}
            </span>
          </div>
        )
      },
    },
    {
      id: "tls",
      header: "TLS",
      cell: ({ row }) => {
        if (row.original.skip_tls_verify) {
          return (
            <div className="flex items-center gap-1 text-xs text-amber-600">
              <AlertCircle className="h-3 w-3" />
              Disabled
            </div>
          )
        }
        return <span className="text-xs text-muted-foreground">Enabled</span>
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center gap-1 justify-end">
          <Button variant="ghost" size="icon-xs" onClick={() => handleOpenEdit(row.original)}>
            <Edit2 className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon-xs" onClick={() => handleOpenDelete(row.original)}>
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <GamepadDirectional className="h-4 w-4" />
            Integrations
          </CardTitle>
          <CardDescription>
            Configure third-party service integrations for this cluster
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : integrations.length === 0 ? (
            <EmptyState
              title="No integrations configured"
              description="Add Prometheus, Grafana, or other integrations to enable monitoring"
              icon={GamepadDirectional}
              actionText="Add Integration"
              onAction={handleOpenCreate}
              actionIcon={Plus}
            />
          ) : (
            <DataTable
              columns={columns}
              data={integrations}
              searchKey="name"
              searchPlaceholder="Filter integrations..."
              toolbarActions={() => (
                <Button onClick={handleOpenCreate}>
                  <Plus />
                  Add Integration
                </Button>
              )}
            />
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-160 max-w-lg">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingIntegration ? "Edit Integration" : "Add Integration"}</DialogTitle>
              <DialogDescription>
                Configure a third-party service integration for this cluster
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="My Prometheus"
                    required
                  />
                </FieldContent>
              </Field>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>Type</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={formData.integration_type}
                      onValueChange={(v) => v && setFormData({ ...formData, integration_type: v as IntegrationType })}
                      disabled={!!editingIntegration}
                      itemToStringLabel={(v) => INTEGRATION_TYPES.find((t) => t.value === v)?.label ?? v ?? ""}
                    >
                      <ComboboxInput />
                      <ComboboxContent>
                        <ComboboxList>
                          {INTEGRATION_TYPES.map((t) => (
                            <ComboboxItem key={t.value} value={t.value}>
                              <div className="flex flex-col">
                                <span>{t.label}</span>
                                <span className="text-muted-foreground text-[10px]">{t.description}</span>
                              </div>
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>Access Mode</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={formData.access_mode}
                      onValueChange={(v) => v && setFormData({ ...formData, access_mode: v as "endpoint" | "service" })}
                      itemToStringLabel={(v) => {
                        if (v === 'endpoint') return 'Endpoint URL';
                        if (v === 'service') return 'Cluster Service Proxy';
                        return v ?? "";
                      }}
                    >
                      <ComboboxInput />
                      <ComboboxContent>
                        <ComboboxList>
                          <ComboboxItem value="endpoint">
                            <div className="flex flex-col">
                              <span>Endpoint URL</span>
                              <span className="text-muted-foreground text-[10px]">Direct external URL endpoint</span>
                            </div>
                          </ComboboxItem>
                          <ComboboxItem value="service">
                            <div className="flex flex-col">
                              <span>Cluster Service Proxy</span>
                              <span className="text-muted-foreground text-[10px]">Route through an in-cluster Kubernetes service</span>
                            </div>
                          </ComboboxItem>
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
              </div>

              {formData.access_mode === "endpoint" ? (
                <Field>
                  <FieldLabel>Endpoint URL</FieldLabel>
                  <FieldContent>
                    <Input
                      value={formData.endpoint}
                      onChange={(e) => setFormData({ ...formData, endpoint: e.target.value })}
                      placeholder="https://prometheus.example.com"
                      type="url"
                      required
                    />
                  </FieldContent>
                </Field>
              ) : (<div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel>Namespace</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={formData.namespace || ""}
                      onValueChange={(v) => setFormData({ ...formData, namespace: v ?? "", service_name: "" })}
                    >
                      <ComboboxInput placeholder="Select namespace" />
                      <ComboboxContent>
                        <ComboboxList>
                          {namespaces.map((ns) => (
                            <ComboboxItem key={ns} value={ns}>
                              {ns}
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
                <div className="flex gap-2">
                  <Field className="flex-1">
                    <FieldLabel>Service Name</FieldLabel>
                    <FieldContent>
                      <Combobox
                        value={formData.service_name || ""}
                        onValueChange={(v) => setFormData({ ...formData, service_name: v ?? "" })}
                        disabled={!formData.namespace}
                      >
                        <ComboboxInput placeholder="Select service" />
                        <ComboboxContent>
                          <ComboboxList>
                            {services.map((svc) => (
                              <ComboboxItem key={svc} value={svc}>
                                {svc}
                              </ComboboxItem>
                            ))}
                          </ComboboxList>
                        </ComboboxContent>
                      </Combobox>
                    </FieldContent>
                  </Field>
                  <Field className="w-24">
                    <FieldLabel>Port</FieldLabel>
                    <FieldContent>
                      <Input
                        value={formData.service_port}
                        onChange={(e) => setFormData({ ...formData, service_port: e.target.value })}
                        placeholder="80"
                        type="number"
                        required
                      />
                    </FieldContent>
                  </Field>
                </div>
              </div>
              )}

              {formData.access_mode === "endpoint" && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <Field>
                      <FieldLabel>Username (optional)</FieldLabel>
                      <FieldContent>
                        <Input
                          value={formData.username}
                          onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                          placeholder="admin"
                        />
                      </FieldContent>
                    </Field>

                    <Field>
                      <FieldLabel>Password (optional)</FieldLabel>
                      <FieldContent>
                        <Input
                          value={formData.password}
                          onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                          type="password"
                          autoComplete="new-password"
                          placeholder={editingIntegration ? "Leave blank to keep current" : ""}
                        />
                      </FieldContent>
                    </Field>
                  </div>

                  <Field>
                    <FieldLabel>Bearer Token (optional)</FieldLabel>
                    <FieldContent>
                      <Input
                        value={formData.token}
                        onChange={(e) => setFormData({ ...formData, token: e.target.value })}
                        placeholder={editingIntegration ? "Leave blank to keep current" : ""}
                      />
                    </FieldContent>
                  </Field>

                  <Field>
                    <FieldLabel>CA Certificate (optional)</FieldLabel>
                    <FieldContent>
                      <Textarea
                        value={formData.ca_cert}
                        onChange={(e) => setFormData({ ...formData, ca_cert: e.target.value })}
                        placeholder="-----BEGIN CERTIFICATE-----..."
                        rows={3}
                      />
                    </FieldContent>
                  </Field>

                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="skip-tls-checkbox"
                      checked={formData.skip_tls_verify}
                      onCheckedChange={(c) => setFormData({ ...formData, skip_tls_verify: !!c })}
                    />
                    <label htmlFor="skip-tls-checkbox" className="cursor-pointer">
                      Skip TLS Verification
                    </label>
                  </div>
                </>
              )}

              <div className="flex items-center gap-2">
                <Checkbox
                  id="enabled-checkbox"
                  checked={formData.enabled}
                  onCheckedChange={(c) => setFormData({ ...formData, enabled: !!c })}
                />
                <label htmlFor="enabled-checkbox" className="cursor-pointer">
                  Enabled
                </label>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleCloseDialog}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {(createMutation.isPending || updateMutation.isPending) && (
                  <Loader2 className="h-4 w-4 animate-spin" />
                )}
                {editingIntegration ? "Save Changes" : "Add Integration"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Integration?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the "{deletingIntegration?.name}" integration.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingIntegration && deleteMutation.mutate(deletingIntegration.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
