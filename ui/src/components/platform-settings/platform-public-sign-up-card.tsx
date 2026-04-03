import { Loader2, Settings2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { usePublicSignUpSettings, useUpdatePublicSignUpSettingsMutation } from "@/hooks/use-platform-settings"

export function PlatformPublicSignUpCard() {
  const { data, isLoading } = usePublicSignUpSettings()
  const updateMutation = useUpdatePublicSignUpSettingsMutation()
  const [enabled, setEnabled] = React.useState(true)

  React.useEffect(() => {
    if (typeof data?.enabled === "boolean") {
      setEnabled(data.enabled)
    }
  }, [data?.enabled])

  const handleToggle = (checked: boolean) => {
    setEnabled(checked)
    updateMutation.mutate(
      { enabled: checked },
      {
        onSuccess: () => {
          toast.success(checked ? "Public registration enabled" : "Public registration disabled")
        },
        onError: (error) => {
          setEnabled(data?.enabled ?? true)
          toast.error("Failed to update public registration", {
            description: error instanceof Error ? error.message : String(error),
          })
        },
      },
    )
  }

  return (
    <Card className="bg-linear-to-b/increasing from-emerald-500/5 to-transparent">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings2 className="h-4 w-4" />
          Public Registration
        </CardTitle>
        <CardDescription>
          Control whether new users can create accounts by email verification.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex items-center justify-between gap-4">
        <div className="space-y-1">
          <p className="text-sm font-medium">
            {enabled ? "Enabled" : "Disabled"}
          </p>
          <p className="text-xs text-muted-foreground">
            When enabled, users must verify their email code before account creation.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {(isLoading || updateMutation.isPending) && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
          <Switch checked={enabled} onCheckedChange={handleToggle} disabled={isLoading || updateMutation.isPending} />
        </div>
      </CardContent>
    </Card>
  )
}
