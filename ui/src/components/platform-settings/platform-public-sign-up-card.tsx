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
    <Card className="bg-linear-to-b/increasing from-emerald-500/5 to-transparent">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings2 className="h-4 w-4" />
          Public Registration
        </CardTitle>
        <CardDescription>
          Control whether new users can create accounts and whether sign-up requires email verification.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div className="space-y-1">
            <p className="text-sm font-medium">
              Public registration: {enabled ? "Enabled" : "Disabled"}
            </p>
            <p className="text-xs text-muted-foreground">
              Allow unauthenticated visitors to create accounts.
            </p>
          </div>
          <div className="flex items-center gap-3">
            {(isLoading || updateMutation.isPending) && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
            <Switch checked={enabled} onCheckedChange={handleEnabledToggle} disabled={isLoading || updateMutation.isPending} />
          </div>
        </div>
        <div className="flex items-center justify-between gap-4">
          <div className="space-y-1">
            <p className="text-sm font-medium">
              Email verification: {emailVerificationRequired ? "Required" : "Optional"}
            </p>
            <p className="text-xs text-muted-foreground">
              When disabled, users can sign up without requesting a verification code.
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
