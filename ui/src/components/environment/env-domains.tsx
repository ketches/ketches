import { AxiosError } from "axios"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { ColumnDef } from "@tanstack/react-table"
import { Clock, Globe, Loader2, Pencil, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { domainsApi, type Domain } from "@/api/domains"
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
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { formatDate } from "@/lib/utils"
import { isPatternDomain, isValidDomainValue, normalizeDomainValue } from "@/lib/domain"

interface EnvDomainsProps {
  envId: string
  isViewer?: boolean
}

interface DomainFormData {
  name: string
  domain: string
  description: string
}

const defaultFormData: DomainFormData = {
  name: "",
  domain: "",
  description: "",
}

export function EnvDomains({ envId, isViewer }: EnvDomainsProps) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [editingItem, setEditingItem] = React.useState<Domain | null>(null)
  const [deletingItem, setDeletingItem] = React.useState<Domain | null>(null)
  const [formData, setFormData] = React.useState<DomainFormData>(defaultFormData)
  const [domainError, setDomainError] = React.useState<string>()

  const { data: response, isLoading } = useQuery({
    queryKey: ["env-domains", envId],
    queryFn: () => domainsApi.listByEnv(envId),
  })
  const items = response?.items ?? []

  const createMutation = useMutation({
    mutationFn: (data: DomainFormData) => domainsApi.createForEnv(envId, {
      ...data,
      domain: normalizeDomainValue(data.domain),
    }),
    onSuccess: () => {
      toast.success("Domain added")
      queryClient.invalidateQueries({ queryKey: ["env-domains", envId] })
      handleCloseDialog()
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to add domain", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: DomainFormData }) => domainsApi.update("env", envId, id, {
      name: data.name,
      description: data.description,
      domain: normalizeDomainValue(data.domain),
    }),
    onSuccess: () => {
      toast.success("Domain updated")
      queryClient.invalidateQueries({ queryKey: ["env-domains", envId] })
      handleCloseDialog()
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to update domain", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => domainsApi.delete("env", envId, id),
    onSuccess: () => {
      toast.success("Domain deleted")
      queryClient.invalidateQueries({ queryKey: ["env-domains", envId] })
      setDeleteOpen(false)
      setDeletingItem(null)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete domain", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleCloseDialog = () => {
    setDialogOpen(false)
    setEditingItem(null)
    setFormData(defaultFormData)
    setDomainError(undefined)
  }

  const validateForm = () => {
    const normalizedDomain = normalizeDomainValue(formData.domain)
    if (!isValidDomainValue(normalizedDomain)) {
      setDomainError("Must be a valid domain such as example.com or *.example.com")
      return false
    }
    setDomainError(undefined)
    return true
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateForm()) {
      return
    }
    if (editingItem) {
      updateMutation.mutate({ id: editingItem.id, data: formData })
      return
    }
    createMutation.mutate(formData)
  }

  const columns: ColumnDef<Domain>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "domain",
      header: "Domain",
      cell: ({ row }) => <span className="font-mono text-sm">{row.original.domain}</span>,
    },
    {
      id: "type",
      header: "Type",
      cell: ({ row }) => <span>{isPatternDomain(row.original.domain) ? "Pattern" : "Standard"}</span>,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.description || "-"}</span>,
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
    ...(!isViewer ? [{
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }: { row: { original: Domain } }) => (
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              setEditingItem(row.original)
              setFormData({
                name: row.original.name,
                domain: row.original.domain,
                description: row.original.description,
              })
              setDomainError(undefined)
              setDialogOpen(true)
            }}
          >
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={() => {
              setDeletingItem(row.original)
              setDeleteOpen(true)
            }}
            disabled={deleteMutation.isPending}
          >
            <Trash2 />
          </Button>
        </div>
      ),
    } as ColumnDef<Domain>] : []),
  ]

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-4 w-4" />
            Domains
          </CardTitle>
          <CardDescription>
            Environment-level domains available to applications in this environment.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={items}
            sourceDataCount={items.length}
            isLoading={isLoading}
            searchKey="name"
            searchPlaceholder="Filter domains..."
            sourceEmptyContent={(
              <EmptyState
                title="No domains configured"
                description="Add domains for application HTTP gateways."
                icon={Globe}
                actionText={isViewer ? undefined : "Add Domain"}
                onAction={isViewer ? undefined : () => {
                  setEditingItem(null)
                  setFormData(defaultFormData)
                  setDomainError(undefined)
                  setDialogOpen(true)
                }}
                actionIcon={isViewer ? undefined : Plus}
              />
            )}
            useStandaloneEmptyState
            rightToolbar={!isViewer ? () => (
              <Button onClick={() => {
                setEditingItem(null)
                setFormData(defaultFormData)
                setDomainError(undefined)
                setDialogOpen(true)
              }}>
                <Plus />
                Add Domain
              </Button>
            ) : undefined}
          />
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingItem ? "Edit Domain" : "Add Domain"}</DialogTitle>
              <DialogDescription>
                Configure domains that application HTTP gateways can reuse.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
                    placeholder="primary-domain"
                    required
                  />
                </FieldContent>
              </Field>
              <Field data-invalid={!!domainError}>
                <FieldLabel>Domain *</FieldLabel>
                <FieldContent>
                  <Input
                    value={formData.domain}
                    onChange={(e) => setFormData((prev) => ({ ...prev, domain: e.target.value }))}
                    placeholder="example.com or *.example.com"
                    aria-invalid={!!domainError}
                    required
                  />
                </FieldContent>
                {domainError && (
                  <FieldError>{domainError}</FieldError>
                )}
              </Field>
              <Field>
                <FieldLabel>Description</FieldLabel>
                <FieldContent>
                  <Textarea
                    value={formData.description}
                    onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                    placeholder="Optional description"
                    className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                  />
                </FieldContent>
              </Field>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleCloseDialog}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {(createMutation.isPending || updateMutation.isPending) && <Loader2 className="h-4 w-4 animate-spin" />}
                {editingItem ? "Save Changes" : "Add Domain"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Domain?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the domain "{deletingItem?.name}".
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingItem && deleteMutation.mutate(deletingItem.id)}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
