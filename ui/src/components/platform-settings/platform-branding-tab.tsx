import * as React from "react"
import { Type } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { usePlatformBranding, useUpdatePlatformBrandingMutation } from "@/hooks/use-platform-settings"

export function PlatformBrandingTab() {
  const { data } = usePlatformBranding()
  const updateMutation = useUpdatePlatformBrandingMutation()
  const [name, setName] = React.useState("")

  React.useEffect(() => {
    setName(data?.name ?? "Ketches Admin")
  }, [data?.name])

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    updateMutation.mutate(
      {
        name: name.trim(),
      },
      {
        onSuccess: () => {
          toast.success("Platform branding updated")
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
    <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Type className="h-4 w-4" />
          Branding
        </CardTitle>
        <CardDescription>
          Configure the platform name shown in the admin shell. The built-in Ketches logo remains
          fixed until image assets move to a dedicated file service.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <Field>
            <FieldLabel htmlFor="platform-name">Platform Name</FieldLabel>
            <FieldContent>
              <Input
                id="platform-name"
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

          <div className="flex justify-end">
            <Button type="submit" disabled={updateMutation.isPending}>
              Save
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
