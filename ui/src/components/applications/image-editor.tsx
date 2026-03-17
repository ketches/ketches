import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Key, Loader2, RefreshCw } from "lucide-react"
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
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import type { AxiosError } from "axios"

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
  const [showCredentials, setShowCredentials] = React.useState(false)
  const [imageOptionsOpen, setImageOptionsOpen] = React.useState(false)

  const [formData, setFormData] = React.useState({
    container_image: "",
    registry_username: "",
    registry_password: "",
  })

  React.useEffect(() => {
    if (app && open) {
      setFormData({
        container_image: app.container_image,
        registry_username: app.registry_username || "",
        registry_password: app.registry_password || "",
      })
      setShowCredentials(Boolean(app.registry_username || app.registry_password))
      setImageOptionsOpen(false)
    }
  }, [app, open])

  const tagsQuery = useQuery({
    queryKey: ["app-image-tags", app?.id],
    queryFn: () => appsApi.listImageTags(app!.id),
    enabled: false,
  })

  const imageOptions = React.useMemo(() => {
    const repository = tagsQuery.data?.repository

    if (!repository) {
      return []
    }

    return (tagsQuery.data?.tags ?? []).map((tag) => {
      const image = `${repository}:${tag}`

      return {
        label: image,
        value: image,
      }
    })
  }, [tagsQuery.data])

  const handleContainerImageChange = React.useCallback((containerImage: string) => {
    setFormData((prev) => ({ ...prev, container_image: containerImage }))
  }, [])

  const handleImageOptionChange = React.useCallback((value: string | null) => {
    if (!value) {
      return
    }

    handleContainerImageChange(value)
    setImageOptionsOpen(false)
  }, [handleContainerImageChange])

  const handleRefreshImageOptions = React.useCallback(async () => {
    if (!app) {
      return
    }

    const result = await tagsQuery.refetch()

    if (result.error) {
      setImageOptionsOpen(false)
      return
    }

    setImageOptionsOpen((result.data?.tags?.length ?? 0) > 0)
  }, [app, tagsQuery])

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
    onError: (error: AxiosError<{ error: string }>) => {
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
      <DialogContent className="sm:max-w-160">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Update Application Image</DialogTitle>
            <DialogDescription>
              Refresh available images from the registry or manually enter a container image.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <div className="flex items-center gap-2">
                <FieldLabel htmlFor="container-image">Container Image *</FieldLabel>
                <div className="flex items-center gap-1 ml-auto">
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          type="button"
                          variant="secondary"
                          size="icon-sm"
                          aria-label="Refresh image options"
                          disabled={tagsQuery.isFetching}
                          onClick={() => {
                            void handleRefreshImageOptions()
                          }}
                        >
                          <RefreshCw className={tagsQuery.isFetching ? "animate-spin" : ""} />
                        </Button>
                      }
                    >
                      <RefreshCw className={tagsQuery.isFetching ? "animate-spin" : ""} />
                    </TooltipTrigger>
                    <TooltipContent>Refresh image options</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          type="button"
                          variant={showCredentials ? "default" : "secondary"}
                          size="icon-sm"
                          aria-label="Registry credentials"
                          aria-pressed={showCredentials}
                          onClick={() => setShowCredentials((prev) => !prev)}
                          className="ml-auto"
                        />
                      }
                    >
                      <Key />
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Registry Credentials</p>
                    </TooltipContent>
                  </Tooltip>
                </div>
              </div>
              <FieldContent>
                {/* <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-1"> */}
                {/* <div className="min-w-0"> */}
                <Combobox
                  open={imageOptionsOpen}
                  onOpenChange={setImageOptionsOpen}
                  value={formData.container_image}
                  onValueChange={handleImageOptionChange}
                  itemToStringLabel={(value) =>
                    imageOptions.find((option) => option.value === value)?.label ?? value ?? ""
                  }
                >
                  <ComboboxInput
                    id="container-image"
                    name="container_image"
                    placeholder="e.g. nginx:latest"
                    value={formData.container_image}
                    onInput={(e) => handleContainerImageChange((e.target as HTMLInputElement).value)}
                    onChange={(e) => handleContainerImageChange(e.target.value)}
                    required
                    className="w-full"
                  />
                  <ComboboxContent>
                    <ComboboxList>
                      {imageOptions.map((option) => (
                        <ComboboxItem key={option.value} value={option.value}>
                          {option.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
                {/* </div> */}

                {/* <div className="flex items-center gap-1">
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant="secondary"
                            size="icon"
                            aria-label="Refresh image options"
                            disabled={tagsQuery.isFetching}
                            onClick={() => {
                              void tagsQuery.refetch()
                            }}
                          >
                            <RefreshCw className={tagsQuery.isFetching ? "animate-spin" : ""} />
                          </Button>
                        }
                      >
                        <RefreshCw className={tagsQuery.isFetching ? "animate-spin" : ""} />
                      </TooltipTrigger>
                      <TooltipContent>Refresh image options</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant={showCredentials ? "default" : "secondary"}
                            size="icon"
                            aria-label="Registry credentials"
                            aria-pressed={showCredentials}
                            onClick={() => setShowCredentials((prev) => !prev)}
                            className="ml-auto"
                          />
                        }
                      >
                        <Key />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Registry Credentials</p>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </div> */}

                {tagsQuery.isLoading && (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground py-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading image options...
                  </div>
                )}
                {tagsQuery.isError && (
                  <p className="text-xs text-destructive py-1">
                    Failed to load image options. You can still enter an image manually.
                  </p>
                )}
              </FieldContent>
            </Field>

            {showCredentials && (
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel htmlFor="registry-username">Registry Username</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry-username"
                      placeholder="Registry username"
                      value={formData.registry_username}
                      onChange={(e) => setFormData((prev) => ({ ...prev, registry_username: e.target.value }))}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel htmlFor="registry-password">Registry Password</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry-password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="Registry password"
                      value={formData.registry_password}
                      onChange={(e) => setFormData((prev) => ({ ...prev, registry_password: e.target.value }))}
                    />
                  </FieldContent>
                </Field>
              </div>
            )}
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
