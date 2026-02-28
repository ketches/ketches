import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { clustersApi } from "@/api/clusters"
import { envsApi } from "@/api/envs"
import { Button } from "@/components/ui/button"
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
import { SimpleCombobox } from "@/components/ui/simple-combobox"

export interface VolumeSpec {
  id?: string
  slug: string
  volume_type: string
  mount_path: string
  sub_path?: string
  storage_class?: string
  capacity?: number
  access_modes?: string
  volume_mode?: string
}

interface VolumeEditorProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  app: App
  volume?: VolumeSpec | null
  onSuccess?: () => void
}

const VOLUME_TYPE_OPTIONS = [
  { value: "pvc", label: "Persistent Storage (PVC)" },
  { value: "emptyDir", label: "Temporary Storage (emptyDir)" },
  { value: "hostPath", label: "Local Storage (hostPath)" },
]

const ACCESS_MODE_OPTIONS = [
  { value: "ReadWriteOnce", label: "Single Node Read/Write (ReadWriteOnce)" },
  { value: "ReadWriteMany", label: "Multi Node Read/Write (ReadWriteMany)" },
  { value: "ReadOnlyMany", label: "Multi Node Read Only (ReadOnlyMany)" },
]

const VOLUME_MODE_OPTIONS = [
  { value: "Filesystem", label: "Filesystem" },
  { value: "Block", label: "Block" },
]

export function VolumeEditor({
  open,
  onOpenChange,
  app,
  volume,
  onSuccess,
}: VolumeEditorProps) {
  const queryClient = useQueryClient()

  const [formData, setFormData] = React.useState<VolumeSpec>({
    slug: "",
    volume_type: "pvc",
    mount_path: "",
    sub_path: "",
    storage_class: "",
    capacity: 10,
    access_modes: "ReadWriteOnce",
    volume_mode: "Filesystem",
  })

  const [errors, setErrors] = React.useState<Record<string, string>>({})

  // Fetch env to get cluster_id
  const { data: env } = useQuery({
    queryKey: ["env", app.env_id],
    queryFn: () => envsApi.get(app.env_id!),
    enabled: !!app.env_id && open && formData.volume_type === "pvc",
  })

  // Fetch storage classes from cluster
  const { data: storageClasses = [] } = useQuery({
    queryKey: ["storage-classes", env?.cluster_id],
    queryFn: async () => {
      if (!env?.cluster_id) return []
      return clustersApi.listStorageClasses(env.cluster_id)
    },
    enabled: !!env?.cluster_id && open && formData.volume_type === "pvc",
  })

  // Reset form when dialog opens/closes or volume changes
  React.useEffect(() => {
    if (open) {
      if (volume) {
        setFormData({
          id: volume.id,
          slug: volume.slug || "",
          volume_type: volume.volume_type || "pvc",
          mount_path: volume.mount_path || "",
          sub_path: volume.sub_path || "",
          storage_class: volume.storage_class || "",
          capacity: volume.capacity || 10,
          access_modes: volume.access_modes || "ReadWriteOnce",
          volume_mode: volume.volume_mode || "Filesystem",
        })
      } else {
        setFormData({
          slug: "",
          volume_type: "pvc",
          mount_path: "",
          sub_path: "",
          storage_class: "",
          capacity: 10,
          access_modes: "ReadWriteOnce",
          volume_mode: "Filesystem",
        })
      }
      setErrors({})
    }
  }, [open, volume])

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {}

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    if (!formData.mount_path.trim()) {
      newErrors.mount_path = "Mount path is required"
    } else if (!formData.mount_path.startsWith("/")) {
      newErrors.mount_path = "Mount path must start with /"
    }

    if (formData.volume_type === "pvc") {
      if (!formData.capacity || formData.capacity < 1) {
        newErrors.capacity = "Capacity must be at least 1 GiB"
      }
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const saveMutation = useMutation({
    mutationFn: async (data: VolumeSpec) => {
      if (data.id) {
        return await appsApi.updateVolume(data.id, data)
      } else {
        return await appsApi.addVolume(app.id, data)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-volumes", app.id] })
      toast.success(
        volume ? "Volume updated successfully" : "Volume added successfully"
      )
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to save volume", {
        description: err.response?.data?.error || "Unknown error",
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!validateForm()) return

    saveMutation.mutate(formData)
  }

  const isPVCType = formData.volume_type === "pvc"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {volume ? "Edit Volume" : "Add Volume"}
            </DialogTitle>
            <DialogDescription>
              Configure persistent or ephemeral storage for your application
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <Field>
              <FieldLabel htmlFor="slug">Slug *</FieldLabel>
              <FieldContent>
                <Input
                  id="slug"
                  value={formData.slug}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, slug: e.target.value }))
                  }
                  placeholder="data-volume"
                  required
                />
              </FieldContent>
              {errors.slug && <FieldError>{errors.slug}</FieldError>}
            </Field>

            <Field>
              <FieldLabel htmlFor="volume-type">Storage Volume Type *</FieldLabel>
              <FieldContent>
                <SimpleCombobox
                  value={formData.volume_type}
                  onValueChange={(value) => value && setFormData((prev) => ({ ...prev, volume_type: value }))}
                  options={VOLUME_TYPE_OPTIONS}
                  className="w-full"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="mount-path">Mount Path *</FieldLabel>
              <FieldContent>
                <Input
                  id="mount-path"
                  value={formData.mount_path}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, mount_path: e.target.value }))
                  }
                  placeholder="/data"
                  required
                />
              </FieldContent>
              {errors.mount_path && <FieldError>{errors.mount_path}</FieldError>}
            </Field>

            <Field>
              <FieldLabel htmlFor="sub-path">SubPath</FieldLabel>
              <FieldContent>
                <Input
                  id="sub-path"
                  value={formData.sub_path}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, sub_path: e.target.value }))
                  }
                  placeholder="Optional"
                />
              </FieldContent>
            </Field>

            {isPVCType && (
              <>
                <Field>
                  <FieldLabel htmlFor="storage-class">Storage Class</FieldLabel>
                  <FieldContent>
                    <SimpleCombobox
                      value={formData.storage_class || "default"}
                      onValueChange={(value) =>
                        value && setFormData((prev) => ({
                          ...prev,
                          storage_class: value === "default" ? "" : value,
                        }))
                      }
                      options={[
                        { value: "default", label: "Default (cluster default)" },
                        ...storageClasses.map((sc: any) => ({
                          value: sc.name,
                          label: sc.name,
                          description: `${sc.provisioner}${sc.isDefault ? " (default)" : ""}`,
                        })),
                      ]}
                      placeholder="Select storage class (default: cluster default)"
                      className="w-full"
                    />
                  </FieldContent>
                </Field>

                <Field>
                  <FieldLabel htmlFor="capacity">Storage Capacity (GiB) *</FieldLabel>
                  <FieldContent>
                    <Input
                      id="capacity"
                      type="number"
                      value={formData.capacity}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          capacity: parseInt(e.target.value, 10) || 10,
                        }))
                      }
                      placeholder="10"
                      min="1"
                      required
                    />
                  </FieldContent>
                  {errors.capacity && <FieldError>{errors.capacity}</FieldError>}
                </Field>

                <Field>
                  <FieldLabel htmlFor="access-modes">Access Mode *</FieldLabel>
                  <FieldContent>
                    <SimpleCombobox
                      value={formData.access_modes}
                      onValueChange={(value) =>
                        value && setFormData((prev) => ({ ...prev, access_modes: value }))
                      }
                      options={ACCESS_MODE_OPTIONS}
                      className="w-full"
                    />
                  </FieldContent>
                </Field>

                <Field>
                  <FieldLabel htmlFor="volume-mode">Storage Mode *</FieldLabel>
                  <FieldContent>
                    <SimpleCombobox
                      value={formData.volume_mode}
                      onValueChange={(value) =>
                        value && setFormData((prev) => ({ ...prev, volume_mode: value }))
                      }
                      options={VOLUME_MODE_OPTIONS}
                      className="w-full"
                    />
                  </FieldContent>
                </Field>
              </>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={saveMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={saveMutation.isPending}>
              {saveMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              {volume ? "Update" : "Add"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
