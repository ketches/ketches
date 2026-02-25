import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2, Pencil, Plus, ShieldCheck, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { certificatesApi, type Certificate } from "@/api/certificates"
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
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"

interface ClusterCertificatesProps {
  clusterId: string
}

interface CertFormData {
  name: string
  description: string
  cert: string
  key: string
}

const defaultFormData: CertFormData = {
  name: "",
  description: "",
  cert: "",
  key: "",
}

export function ClusterCertificates({ clusterId }: ClusterCertificatesProps) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [editingCert, setEditingCert] = React.useState<Certificate | null>(null)
  const [deletingCert, setDeletingCert] = React.useState<Certificate | null>(null)
  const [formData, setFormData] = React.useState<CertFormData>(defaultFormData)

  const { data: certsResponse, isLoading } = useQuery({
    queryKey: ["cluster-certificates", clusterId],
    queryFn: () => certificatesApi.listByCluster(clusterId),
  })
  const certificates = certsResponse?.items ?? []

  const createMutation = useMutation({
    mutationFn: (data: CertFormData) =>
      certificatesApi.createForCluster(clusterId, { ...data, scope: "cluster" }),
    onSuccess: () => {
      toast.success("Certificate added")
      queryClient.invalidateQueries({ queryKey: ["cluster-certificates", clusterId] })
      handleCloseDialog()
    },
    onError: (error: any) => {
      toast.error("Failed to add certificate", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CertFormData }) => {
      const payload: { name?: string; description?: string; cert?: string; key?: string } = {
        name: data.name,
        description: data.description,
      }
      if (data.cert) payload.cert = data.cert
      if (data.key) payload.key = data.key
      return certificatesApi.update("cluster", clusterId, id, payload)
    },
    onSuccess: () => {
      toast.success("Certificate updated")
      queryClient.invalidateQueries({ queryKey: ["cluster-certificates", clusterId] })
      handleCloseDialog()
    },
    onError: (error: any) => {
      toast.error("Failed to update certificate", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => certificatesApi.delete("cluster", clusterId, id),
    onSuccess: () => {
      toast.success("Certificate deleted")
      queryClient.invalidateQueries({ queryKey: ["cluster-certificates", clusterId] })
      setDeleteOpen(false)
      setDeletingCert(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete certificate", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleOpenCreate = () => {
    setEditingCert(null)
    setFormData(defaultFormData)
    setDialogOpen(true)
  }

  const handleOpenEdit = (cert: Certificate) => {
    setEditingCert(cert)
    setFormData({
      name: cert.name,
      description: cert.description,
      cert: "",
      key: "",
    })
    setDialogOpen(true)
  }

  const handleCloseDialog = () => {
    setDialogOpen(false)
    setEditingCert(null)
    setFormData(defaultFormData)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingCert) {
      updateMutation.mutate({ id: editingCert.id, data: formData })
    } else {
      createMutation.mutate(formData)
    }
  }

  const handleOpenDelete = (cert: Certificate) => {
    setDeletingCert(cert)
    setDeleteOpen(true)
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-sm flex items-center gap-2">
                <ShieldCheck className="h-4 w-4" />
                TLS Certificates
              </CardTitle>
              <CardDescription>
                Cluster-level certificates available to all environments
              </CardDescription>
            </div>
            <Button size="sm" onClick={handleOpenCreate}>
              <Plus />
              Add Certificate
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : !certificates || certificates.length === 0 ? (
            <EmptyState
              title="No certificates configured"
              description="Add a TLS certificate to enable HTTPS gateways"
              icon={ShieldCheck}
            />
          ) : (
            <div className="border-y border-x-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Created At</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {certificates.map((cert) => (
                    <TableRow key={cert.id}>
                      <TableCell className="font-medium">{cert.name}</TableCell>
                      <TableCell className="text-muted-foreground">{cert.description || "-"}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {new Date(cert.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center gap-1 justify-end">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleOpenEdit(cert)}
                          >
                            <Pencil />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-destructive hover:text-destructive hover:bg-destructive/10"
                            onClick={() => handleOpenDelete(cert)}
                            disabled={deleteMutation.isPending}
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editingCert ? "Edit Certificate" : "Add Certificate"}</DialogTitle>
            <DialogDescription>
              {editingCert
                ? "Update the certificate details. Leave PEM fields blank to keep current values."
                : "Add a TLS certificate to enable HTTPS gateways in this cluster."}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit}>
            <div className="grid gap-4 py-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="my-tls-cert"
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Description</FieldLabel>
                <FieldContent>
                  <Input
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="Optional description"
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>{editingCert ? "Certificate PEM (leave blank to keep current)" : "Certificate PEM *"}</FieldLabel>
                <FieldContent>
                  <Textarea
                    value={formData.cert}
                    onChange={(e) => setFormData({ ...formData, cert: e.target.value })}
                    placeholder="-----BEGIN CERTIFICATE-----&#10;..."
                    rows={6}
                    required={!editingCert}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>{editingCert ? "Private Key PEM (leave blank to keep current)" : "Private Key PEM *"}</FieldLabel>
                <FieldContent>
                  <Textarea
                    value={formData.key}
                    onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                    placeholder="-----BEGIN PRIVATE KEY-----&#10;..."
                    rows={6}
                    required={!editingCert}
                  />
                </FieldContent>
              </Field>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleCloseDialog}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {(createMutation.isPending || updateMutation.isPending) && (
                  <Loader2 className="h-4 w-4 animate-spin" />
                )}
                {editingCert ? "Save Changes" : "Add Certificate"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Certificate?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the certificate "{deletingCert?.name}".
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingCert && deleteMutation.mutate(deletingCert.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
