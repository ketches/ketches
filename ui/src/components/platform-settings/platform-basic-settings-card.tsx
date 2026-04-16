import { Loader2, LogIn, Mail, Settings2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { usePublicSignUpSettings as useBasicSettings, useUpdateBasicSettingsMutation } from "@/hooks/use-platform-settings"

export function PlatformBasicSettingsCard() {
  const { data, isLoading } = useBasicSettings()
  const updateMutation = useUpdateBasicSettingsMutation()
  const [enabled, setEnabled] = React.useState(true)
  const [emailVerificationRequired, setEmailVerificationRequired] = React.useState(true)

  React.useEffect(() => {
    if (typeof data?.enabled === "boolean") {
      setEnabled(data.enabled)
    }
    if (typeof data?.email_verification_required === "boolean") {
      setEmailVerificationRequired(data.email_verification_required)
    }
  }, [data?.email_verification_required, data?.enabled])

  const handleUpdate = (nextEnabled: boolean, nextEmailVerificationRequired: boolean) => {
    updateMutation.mutate(
      {
        enabled: nextEnabled,
        email_verification_required: nextEmailVerificationRequired,
      },
      {
        onSuccess: () => {
          toast.success(nextEnabled ? "Public registration enabled" : "Public registration disabled")
        },
        onError: (error) => {
          setEnabled(data?.enabled ?? true)
          setEmailVerificationRequired(data?.email_verification_required ?? true)
          toast.error("Failed to update public registration", {
            description: error instanceof Error ? error.message : String(error),
          })
        },
      },
    )
  }

  const handleEnabledToggle = (checked: boolean) => {
    setEnabled(checked)
    handleUpdate(checked, emailVerificationRequired)
  }

  const handleEmailVerificationToggle = (checked: boolean) => {
    setEmailVerificationRequired(checked)
    handleUpdate(enabled, checked)
  }

  return (
    <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings2 className="h-4 w-4" />
          Basic Settings
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div className="space-y-1">
            <p className="inline-flex items-center gap-1 text-[12px] font-medium">
              <LogIn className="mr-1 h-4 w-4" />
              Public registration
            </p>
            <p className="text-xs font-medium text-muted-foreground">
              If enabled, anyone can register an account without invitation. Otherwise, only administrators can create accounts for users.
            </p>
          </div>
          <div className="flex items-center gap-3">
            {(isLoading || updateMutation.isPending) && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
            <Switch checked={enabled} onCheckedChange={handleEnabledToggle} disabled={isLoading || updateMutation.isPending} />
          </div>
        </div>
        <Separator orientation="horizontal" className="h-4" />
        <div className="flex items-center justify-between gap-4">
          <div className="space-y-1">
            <p className="inline-flex items-center gap-1 text-[12px] font-medium">
              <Mail className="mr-1 h-4 w-4" />
              Email verification
            </p>
            <p className="text-xs font-medium text-muted-foreground">
              Require users to use email verification code to register an account or reset password and so on.
            </p>
          </div>
          <Switch
            checked={emailVerificationRequired}
            onCheckedChange={handleEmailVerificationToggle}
            disabled={isLoading || updateMutation.isPending}
          />
        </div>
      </CardContent>
    </Card>
  )
}
