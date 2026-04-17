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

interface AddExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddExtensionDialog({
  open,
  onOpenChange,
}: AddExtensionDialogProps) {
  const queryClient = useQueryClient()

  const [name, setName] = React.useState("")
  const [displayName, setDisplayName] = React.useState("")
  const [description, setDescription] = React.useState("")
  const [ociUrl, setOciUrl] = React.useState("")
  const [iconUrl, setIconUrl] = React.useState("")
  const [capabilities, setCapabilities] = React.useState("")
  const [metadata, setMetadata] = React.useState("{}")

  // Reset form when dialog closes
  React.useEffect(() => {
    if (!open) {
      setName("")
      setDisplayName("")
      setDescription("")
      setOciUrl("")
      setIconUrl("")
      setCapabilities("")
      setMetadata("{}")
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

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    const data: CreateExtensionRequest = {
      name: name.trim(),
      oci_url: ociUrl.trim(),
    }
    if (displayName.trim()) data.display_name = displayName.trim()
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
      <DialogContent className="sm:max-w-md">
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
              <FieldLabel htmlFor="ext-name">Name *</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-name"
                  placeholder="e.g. gateway-api"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="ext-display-name">Display Name</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-display-name"
                  placeholder="e.g. Gateway API"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="ext-oci-url">OCI URL *</FieldLabel>
              <FieldContent>
                <Input
                  id="ext-oci-url"
                  placeholder="e.g. oci://docker.io/envoyproxy/gateway-helm"
                  value={ociUrl}
                  onChange={(e) => setOciUrl(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>

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
