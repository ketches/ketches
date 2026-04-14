import { useMutation, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type Extension,
  type UpdateExtensionRequest,
} from "@/api/clusters"
import { Button } from "@/components/ui/button"
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
import { Textarea } from "@/components/ui/textarea"

interface EditExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  item: Extension | null
}

export function EditExtensionDialog({
  open,
  onOpenChange,
  item,
}: EditExtensionDialogProps) {
  const queryClient = useQueryClient()

  const [displayName, setDisplayName] = React.useState("")
  const [description, setDescription] = React.useState("")
  const [ociUrl, setOciUrl] = React.useState("")
  const [iconUrl, setIconUrl] = React.useState("")

  // Populate form fields when item changes
  React.useEffect(() => {
    if (item && open) {
      setDisplayName(item.display_name ?? "")
      setDescription(item.description ?? "")
      setOciUrl(item.oci_url)
      setIconUrl(item.icon_url ?? "")
    }
  }, [item, open])

  // Reset form when dialog closes
  React.useEffect(() => {
    if (!open) {
      setDisplayName("")
      setDescription("")
      setOciUrl("")
      setIconUrl("")
    }
  }, [open])

  const updateMutation = useMutation({
    mutationFn: (data: UpdateExtensionRequest) =>
      clustersApi.updateExtension(item!.id, data),
    onSuccess: () => {
      toast.success("Extension updated")
      queryClient.invalidateQueries({ queryKey: ["extensions"] })
      onOpenChange(false)
    },
    onError: (error: unknown) => {
      const msg =
        error && typeof error === "object" && "response" in error
          ? (
            error as {
              response?: { data?: { error?: string } }
            }
          ).response?.data?.error
          : null
      toast.error("Failed to update extension", {
        description: msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    const data: UpdateExtensionRequest = {
      oci_url: ociUrl.trim() || undefined,
    }
    if (displayName.trim()) data.display_name = displayName.trim()
    if (description.trim()) data.description = description.trim()
    data.icon_url = iconUrl.trim()
    updateMutation.mutate(data)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit} className="flex flex-col gap-0">
          <DialogHeader className="pb-4">
            <DialogTitle>Edit Extension</DialogTitle>
            <DialogDescription>
              Update the metadata for{" "}
              <span className="font-medium">{item?.display_name || item?.name}</span>.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-4 py-2">
            <Field>
              <FieldLabel htmlFor="edit-ext-display-name">Display Name</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-ext-display-name"
                  placeholder="e.g. Gateway API"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="edit-ext-oci-url">OCI URL *</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-ext-oci-url"
                  placeholder="e.g. oci://docker.io/envoyproxy/gateway-helm"
                  value={ociUrl}
                  onChange={(e) => setOciUrl(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="edit-ext-description">Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="edit-ext-description"
                  placeholder="Brief description of what this extension does"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="edit-ext-icon-url">Icon URL</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-ext-icon-url"
                  placeholder="https://example.com/icon.svg"
                  value={iconUrl}
                  onChange={(e) => setIconUrl(e.target.value)}
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter className="pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
