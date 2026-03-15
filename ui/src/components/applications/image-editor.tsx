import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2, RefreshCw } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi, type App } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface ImageEditorProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  app: App | null
  onSuccess?: () => void
}

export function ImageEditor({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  app,
  onSuccess,
}: ImageEditorProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()

  const [formData, setFormData] = React.useState({
    container_image: "",
    registry_username: "",
    registry_password: "",
  })

  const [selectedTag, setSelectedTag] = React.useState("")

  React.useEffect(() => {
    if (app && open) {
      setFormData({
        container_image: app.container_image,
        registry_username: app.registry_username || "",
        registry_password: app.registry_password || "",
      })
      setSelectedTag("")
    }
  }, [app, open])

  const tagsQuery = useQuery({
    queryKey: ["app-image-tags", app?.id],
    queryFn: () => appsApi.listImageTags(app!.id),
    enabled: !!app && open,
  })

  React.useEffect(() => {
    if (tagsQuery.data?.current_tag && !selectedTag) {
      setSelectedTag(tagsQuery.data.current_tag)
    }
  }, [tagsQuery.data, selectedTag])

  const tagOptions = React.useMemo(() => {
    return (tagsQuery.data?.tags ?? []).map((tag) => ({ label: tag, value: tag }))
  }, [tagsQuery.data?.tags])

  const handleTagChange = React.useCallback((value: string | null) => {
    if (!value) {
      setSelectedTag("")
      return
    }

    setSelectedTag(value)
    const repo = tagsQuery.data?.repository
    if (repo) {
      setFormData((prev) => ({ ...prev, container_image: `${repo}:${value}` }))
    }
  }, [tagsQuery.data?.repository])

  const mutation = useMutation({
    mutationFn: (data: { container_image: string; registry_username?: string; registry_password?: string }) => {
      if (!app) throw new Error("No application selected")
      return appsApi.updateImage(app.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app?.id] })
      toast.success("Application image updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update application image",
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-140">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Update Application Image</DialogTitle>
            <DialogDescription>
              Select a tag from the registry or manually enter a container image.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>Container Image *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="e.g. nginx:latest"
                  value={formData.container_image}
                  onChange={(e) => setFormData((prev) => ({ ...prev, container_image: e.target.value }))}
                  required
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel className="flex items-center justify-between">
                <span>Image Tag</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  disabled={tagsQuery.isFetching}
                  onClick={() => void tagsQuery.refetch()}
                >
                  <RefreshCw className={tagsQuery.isFetching ? "animate-spin" : ""} />
                </Button>
              </FieldLabel>
              <FieldContent>
                {tagsQuery.isLoading ? (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground py-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading tags...
                  </div>
                ) : tagsQuery.isError ? (
                  <p className="text-xs text-destructive py-1">
                    Failed to load tags. You can still enter an image manually below.
                  </p>
                ) : (
                  <Combobox
                    value={selectedTag}
                    onValueChange={handleTagChange}
                    itemToStringLabel={(value) => tagOptions.find((o) => o.value === value)?.label ?? value ?? ""}
                  >
                    <ComboboxInput
                      name="image-tag"
                      placeholder="Select or search tag..."
                      value={selectedTag}
                      onInput={(e) => setSelectedTag((e.target as HTMLInputElement).value)}
                      onChange={(e) => setSelectedTag(e.target.value)}
                    />
                    <ComboboxContent>
                      <ComboboxList>
                        {tagOptions.map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            {option.label}
                            {option.value === tagsQuery.data?.current_tag && (
                              <span className="ml-auto text-xs text-muted-foreground">current</span>
                            )}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                )}
              </FieldContent>
            </Field>

            <div className="grid gap-4">
              <Field>
                <FieldLabel>Registry Username</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="Registry username"
                    value={formData.registry_username}
                    onChange={(e) => setFormData((prev) => ({ ...prev, registry_username: e.target.value }))}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Registry Password</FieldLabel>
                <FieldContent>
                  <Input
                    type="password"
                    autoComplete="new-password"
                    placeholder="Registry password"
                    value={formData.registry_password}
                    onChange={(e) => setFormData((prev) => ({ ...prev, registry_password: e.target.value }))}
                  />
                </FieldContent>
              </Field>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  Upgrading...
                </>
              ) : (
                "Upgrade"
              )}
            </Button>
          </DialogFooter>
        </form >
      </DialogContent >
    </Dialog >
  )
}
