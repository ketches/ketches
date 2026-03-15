import * as React from "react"
import { Bell, RefreshCw, Settings2 } from "lucide-react"

import { PlatformUpdateConfigDialog } from "@/components/platform-updates/platform-update-config-dialog"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { getAppBuildTime, getAppBuildVersion } from "@/lib/build-info"
import { usePlatformUpdateStatusQuery, useTriggerPlatformRolloutMutation } from "@/hooks/use-platform-update"
import { useAuthStore } from "@/stores/auth"

const EMPTY_OPTIONS: Array<{ label: string; value: string }> = []

export function PlatformUpdateBell() {
  const role = useAuthStore((state) => state.user?.role)
  const isAdmin = role === "admin"
  const statusQuery = usePlatformUpdateStatusQuery(isAdmin)
  const { data: status } = statusQuery
  const rolloutMutation = useTriggerPlatformRolloutMutation()

  const [configDialogOpen, setConfigDialogOpen] = React.useState(false)
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
    if (!status) return
    const fallbackVersion =
      status.recommended_shared_version ||
      status.api.latest_version ||
      status.ui.latest_version ||
      ""
    if (!apiVersionRef.current) {
      updateApiVersion(fallbackVersion)
    }
    if (!uiVersionRef.current) {
      updateUiVersion(fallbackVersion)
    }
  }, [status, updateApiVersion, updateUiVersion])

  if (!isAdmin) {
    return null
  }

  const apiOptions = status?.api.available_versions?.map((value) => ({ label: value, value })) ?? EMPTY_OPTIONS
  const uiOptions = status?.ui.available_versions?.map((value) => ({ label: value, value })) ?? EMPTY_OPTIONS
  const hasUpdates = Boolean(status?.api.update_available || status?.ui.update_available)
  const rolloutDisabled = !status?.can_rollout || rolloutMutation.isPending || !apiVersion || !uiVersion
  const refreshDisabled = statusQuery.isFetching || rolloutMutation.isPending

  const handleRollout = () => {
    const nextApiVersion = apiVersionRef.current || apiVersion
    const nextUiVersion = uiVersionRef.current || uiVersion
    const payload = nextApiVersion === nextUiVersion
      ? { shared_version: nextApiVersion }
      : { api_version: nextApiVersion, ui_version: nextUiVersion }

    rolloutMutation.mutate(payload)
  }

  return (
    <>
      <Popover>
        <PopoverTrigger>
          <Button
            variant="ghost"
            size="icon-sm"
            className="relative"
            data-testid="platform-update-bell"
          >
            <Bell className="size-4" />
            {hasUpdates && (
              <span className="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-amber-500" />
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-96 space-y-4">
          <div className="space-y-1">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">Platform Updates</p>
              {status?.recommended_shared_version ? (
                <span className="text-xs text-muted-foreground">
                  Recommended {status.recommended_shared_version}
                </span>
              ) : null}
            </div>
            <p className="text-xs text-muted-foreground">
              Local build {getAppBuildVersion()} {getAppBuildTime() ? `(${getAppBuildTime()})` : ""}
            </p>
            <p className="text-xs text-muted-foreground">
              API {status?.api.current_version || "unknown"} / UI {status?.ui.current_version || "unknown"}
            </p>
            {!status?.can_rollout && status?.rollout_blockers?.length ? (
              <p className="text-xs text-amber-700">
                {status.rollout_blockers.join("; ")}
              </p>
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
                    name="api-version"
                    placeholder="Select API version"
                    value={apiVersion}
                    onInput={(event) => updateApiVersion((event.target as HTMLInputElement).value)}
                    onChange={(event) => updateApiVersion(event.target.value)}
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
                    name="ui-version"
                    placeholder="Select UI version"
                    value={uiVersion}
                    onInput={(event) => updateUiVersion((event.target as HTMLInputElement).value)}
                    onChange={(event) => updateUiVersion(event.target.value)}
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

          <div className="flex items-center justify-between gap-2">
            <Button type="button" variant="outline" onClick={() => setConfigDialogOpen(true)}>
              <Settings2 />
              Configure
            </Button>
            <div className="flex items-center gap-2">
              <Button type="button" variant="outline" disabled={refreshDisabled} onClick={() => void statusQuery.refetch()}>
                <RefreshCw />
                Refresh
              </Button>
              <Button
                type="button"
                data-role="rollout-submit"
                disabled={rolloutDisabled}
                onClick={handleRollout}
              >
                Update
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>

      <PlatformUpdateConfigDialog
        open={configDialogOpen}
        onOpenChange={setConfigDialogOpen}
      />
    </>
  )
}
