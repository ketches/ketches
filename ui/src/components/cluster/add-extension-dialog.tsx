import { useMutation, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type CreateExtensionRequest,
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
import { deriveExtensionDefaults, toExtensionSlug } from "./extension-dialog.utils"

interface AddExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddExtensionDialog({
  open,
  onOpenChange,
}: AddExtensionDialogProps) {
  const queryClient = useQueryClient()

  const [slug, setSlug] = React.useState("")
  const [extensionName, setExtensionName] = React.useState("")
  const [description, setDescription] = React.useState("")
  const [ociUrl, setOciUrl] = React.useState("")
  const [iconUrl, setIconUrl] = React.useState("")
  const [capabilities, setCapabilities] = React.useState("")
  const [metadata, setMetadata] = React.useState("{}")
  const [hasEditedName, setHasEditedName] = React.useState(false)
  const [hasEditedSlug, setHasEditedSlug] = React.useState(false)

  // Reset form when dialog closes
  React.useEffect(() => {
    if (!open) {
      setSlug("")
      setExtensionName("")
      setDescription("")
      setOciUrl("")
      setIconUrl("")
      setCapabilities("")
      setMetadata("{}")
      setHasEditedName(false)
      setHasEditedSlug(false)
    }
  }, [open])

  const createMutation = useMutation({
    mutationFn: (data: CreateExtensionRequest) =>
      clustersApi.createExtension(data),
    onSuccess: () => {
      toast.success("Extension added")
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
      toast.error("Failed to add extension", {
        description: msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleOciUrlChange = (value: string) => {
    const defaults = deriveExtensionDefaults(value)
    setOciUrl(value)
    setExtensionName((current) => hasEditedName ? current : defaults.name)
    setSlug((current) => hasEditedSlug ? current : defaults.slug)
  }

  const handleNameChange = (value: string) => {
    setHasEditedName(true)
    setExtensionName(value)
    if (!hasEditedSlug) {
      setSlug(toExtensionSlug(value))
    }
  }

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    const data: CreateExtensionRequest = {
      slug: slug.trim(),
      name: extensionName.trim(),
      oci_url: ociUrl.trim(),
    }
    if (description.trim()) data.description = description.trim()
    if (capabilities.trim()) data.capabilities = capabilities.split(",").map((item) => item.trim()).filter(Boolean)
    try {
      const parsedMetadata = JSON.parse(metadata) as Record<string, unknown>
      if (Object.keys(parsedMetadata).length > 0) data.metadata = parsedMetadata
    } catch {
      toast.error("Invalid extension metadata", { description: "Metadata must be valid JSON." })
      return
    }
    if (iconUrl.trim()) data.icon_url = iconUrl.trim()
    createMutation.mutate(data)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit} className="flex flex-col gap-0">
          <DialogHeader className="pb-4">
            <DialogTitle>Add Extension</DialogTitle>
            <DialogDescription>
              Add a new OCI-based extension. Users can
              then install it on their clusters.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-4 py-2">
            <Field>
              <FieldLabel htmlFor="ext-oci-url">OCI URL *</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-oci-url"
                  name="oci_url"
                  placeholder="e.g. oci://ghcr.io/nginx/charts/nginx-gateway-fabric"
                  value={ociUrl}
                  onChange={(e) => handleOciUrlChange(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="ext-display-name">Name</FieldLabel>
                <FieldContent>
                  <Input
                    id="ext-display-name"
                    name="name"
                    placeholder="e.g. Nginx Gateway Fabric"
                    value={extensionName}
                    onChange={(e) => handleNameChange(e.target.value)}
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="ext-name">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="ext-name"
                    name="slug"
                    placeholder="e.g. nginx-gateway-fabric"
                    value={slug}
                    onChange={(e) => {
                      setHasEditedSlug(true)
                      setSlug(e.target.value)
                    }}
                    required
                  />
                </FieldContent>
              </Field>
            </div>

            <Field>
              <FieldLabel htmlFor="ext-description">Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="ext-description"
                  placeholder="Brief description of what this extension does"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="ext-capabilities">Capabilities</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-capabilities"
                  placeholder="e.g. gateway-api, observability"
                  value={capabilities}
                  onChange={(e) => setCapabilities(e.target.value)}
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="ext-metadata">Metadata (JSON)</FieldLabel>
              <FieldContent>
                <Textarea
                  id="ext-metadata"
                  placeholder='{"gateway_api":{"controller_name":"gateway.envoyproxy.io/gatewayclass-controller"}}'
                  value={metadata}
                  onChange={(e) => setMetadata(e.target.value)}
                  className="min-h-24 max-h-48 resize-y break-all whitespace-pre-wrap font-mono text-xs"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="ext-icon-url">Icon URL</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-icon-url"
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
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? "Adding..." : "Add Extension"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
