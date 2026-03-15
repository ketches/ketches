import * as React from "react"
import { RefreshCw } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { usePlatformUpdateStatusQuery, useTriggerPlatformRolloutMutation } from "@/hooks/use-platform-update"
import { getAppBuildTime, getAppBuildVersion } from "@/lib/build-info"

const EMPTY_OPTIONS: Array<{ label: string; value: string }> = []

interface PlatformUpdateRolloutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onRolloutSuccess?: () => void
}

export function PlatformUpdateRolloutDialog({
  open,
  onOpenChange,
  onRolloutSuccess,
}: PlatformUpdateRolloutDialogProps) {
  const statusQuery = usePlatformUpdateStatusQuery(open)
  const rolloutMutation = useTriggerPlatformRolloutMutation()
  const [apiVersion, setApiVersion] = React.useState("")
  const [uiVersion, setUiVersion] = React.useState("")
  const apiVersionRef = React.useRef("")
  const uiVersionRef = React.useRef("")

  const updateApiVersion = React.useCallback((value: string) => {
    apiVersionRef.current = value
    setApiVersion(value)
  }, [])

  const updateUiVersion = React.useCallback((value: string) => {
    uiVersionRef.current = value
    setUiVersion(value)
  }, [])

  React.useEffect(() => {
    if (!open || !statusQuery.data) return
    const fallbackVersion =
      statusQuery.data.recommended_shared_version ||
      statusQuery.data.api.latest_version ||
      statusQuery.data.ui.latest_version ||
      ""
    updateApiVersion(fallbackVersion)
    updateUiVersion(fallbackVersion)
  }, [open, statusQuery.data, updateApiVersion, updateUiVersion])

  const status = statusQuery.data
  const apiOptions = status?.api.available_versions?.map((value) => ({ label: value, value })) ?? EMPTY_OPTIONS
  const uiOptions = status?.ui.available_versions?.map((value) => ({ label: value, value })) ?? EMPTY_OPTIONS
  const rolloutDisabled = !status?.can_rollout || rolloutMutation.isPending || !apiVersion || !uiVersion

  const handleRollout = () => {
    const nextAPIVersion = apiVersionRef.current || apiVersion
    const nextUIVersion = uiVersionRef.current || uiVersion
    const payload = nextAPIVersion === nextUIVersion
      ? { shared_version: nextAPIVersion }
      : { api_version: nextAPIVersion, ui_version: nextUIVersion }

    rolloutMutation.mutate(payload, {
      onSuccess: () => {
        toast.success("Platform rollout submitted")
        onOpenChange(false)
        onRolloutSuccess?.()
      },
      onError: (error) => {
        toast.error("Failed to submit platform rollout", {
          description: error instanceof Error ? error.message : String(error),
        })
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Platform Updates</DialogTitle>
          <DialogDescription>
            Review the latest platform versions and trigger a Kubernetes rollout when the
            current deployment targets are ready.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1 text-sm">
            <p className="text-muted-foreground">
              Local build {getAppBuildVersion()} {getAppBuildTime() ? `(${getAppBuildTime()})` : ""}
            </p>
            <p className="text-muted-foreground">
              API {status?.api.current_version || "unknown"} / UI {status?.ui.current_version || "unknown"}
            </p>
            {status?.recommended_shared_version ? (
              <p className="font-medium">Recommended {status.recommended_shared_version}</p>
            ) : null}
            {!status?.can_rollout && status?.rollout_blockers?.length ? (
              <p className="text-amber-700">{status.rollout_blockers.join("; ")}</p>
            ) : null}
          </div>

          <div className="space-y-3">
            <Field>
              <FieldLabel>API Version</FieldLabel>
              <FieldContent>
                <Combobox
                  value={apiVersion}
                  onValueChange={(value) => value && updateApiVersion(value)}
                  itemToStringLabel={(value) => apiOptions.find((option) => option.value === value)?.label ?? value ?? ""}
                >
                  <ComboboxInput
                    name="dialog-api-version"
                    value={apiVersion}
                    onInput={(event) => updateApiVersion((event.target as HTMLInputElement).value)}
                    onChange={(event) => updateApiVersion(event.target.value)}
                    placeholder="Select API version"
                  />
                  <ComboboxContent>
                    <ComboboxList>
                      {apiOptions.map((option) => (
                        <ComboboxItem key={option.value} value={option.value}>
                          {option.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>UI Version</FieldLabel>
              <FieldContent>
                <Combobox
                  value={uiVersion}
                  onValueChange={(value) => value && updateUiVersion(value)}
                  itemToStringLabel={(value) => uiOptions.find((option) => option.value === value)?.label ?? value ?? ""}
                >
                  <ComboboxInput
                    name="dialog-ui-version"
                    value={uiVersion}
                    onInput={(event) => updateUiVersion((event.target as HTMLInputElement).value)}
                    onChange={(event) => updateUiVersion(event.target.value)}
                    placeholder="Select UI version"
                  />
                  <ComboboxContent>
                    <ComboboxList>
                      {uiOptions.map((option) => (
                        <ComboboxItem key={option.value} value={option.value}>
                          {option.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => void statusQuery.refetch()} disabled={statusQuery.isFetching || rolloutMutation.isPending}>
            <RefreshCw />
            Refresh
          </Button>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={handleRollout} disabled={rolloutDisabled}>
            Update
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
