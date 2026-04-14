import { Edit2, Loader2, Type } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { usePlatformBranding, useUpdatePlatformBrandingMutation } from "@/hooks/use-platform-settings"

const DEFAULT_PLATFORM_NAME = "Ketches Admin"

export function PlatformBrandingTab() {
  const { data, isLoading } = usePlatformBranding()
  const updateMutation = useUpdatePlatformBrandingMutation()
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [name, setName] = React.useState("")

  React.useEffect(() => {
    if (dialogOpen) {
      setName(data?.name ?? DEFAULT_PLATFORM_NAME)
    }
  }, [data?.name, dialogOpen])

  const handleSubmit = (event: React.SubmitEvent<HTMLFormElement>) => {
    event.preventDefault()

    const trimmedName = name.trim()
    if (!trimmedName) {
      toast.error("Branding name is required")
      return
    }

    updateMutation.mutate(
      {
        name: trimmedName,
      },
      {
        onSuccess: () => {
          toast.success("Platform branding updated")
          setDialogOpen(false)
        },
        onError: (error) => {
          toast.error("Failed to update platform branding", {
            description: error instanceof Error ? error.message : String(error),
          })
        },
      },
    )
  }

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-emerald-500/5 to-transparent group/card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Type className="h-4 w-4" />
            Branding
          </CardTitle>
          <CardDescription>
            Configure the platform name shown in the admin shell.
          </CardDescription>
          <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label="Edit branding name"
                    disabled={isLoading}
                    onClick={() => setDialogOpen(true)}
                  />
                }
              >
                <Edit2 />
              </TooltipTrigger>
              <TooltipContent>Edit</TooltipContent>
            </Tooltip>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Platform Name</p>
              <div className="flex items-center gap-2">
                <p className="text-sm font-mono truncate">{data?.name ?? DEFAULT_PLATFORM_NAME}</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            <DialogHeader>
              <DialogTitle>Edit Platform Branding</DialogTitle>
              <DialogDescription>
                Update the platform name shown in the sidebar header and platform settings page.
              </DialogDescription>
            </DialogHeader>

            <Field>
              <FieldLabel htmlFor="platform-name">Branding Name</FieldLabel>
              <FieldContent>
                <Input
                  id="platform-name"
                  name="platform-name"
                  type="text"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  disabled={updateMutation.isPending}
                  required
                />
                <FieldDescription>
                  This name appears in the sidebar header and on the platform settings page.
                </FieldDescription>
              </FieldContent>
            </Field>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  "Save"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
