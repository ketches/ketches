import { DiffEditor, Editor, type MonacoDiffEditor } from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { GitCompare, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ClusterExtension,
  type ExtensionVersionInfo,
  type UpgradeExtensionRequest,
} from "@/api/clusters"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { filterSelectableExtensionVersions } from "@/lib/extension-versions"

interface UpdateExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  extension: ClusterExtension | null
}

export function UpdateExtensionDialog({
  open,
  onOpenChange,
  clusterId,
  extension,
}: UpdateExtensionDialogProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [selectedVersion, setSelectedVersion] = React.useState("")
  const [modifiedValues, setModifiedValues] = React.useState("")
  const [showDiff, setShowDiff] = React.useState(false)
  const isRetryingFailedUpdate = extension?.status === "failed" && extension.phase === "upgrading"

  // Resolved theme for Monaco
  const [monacoTheme, setMonacoTheme] = React.useState<"vs" | "vs-dark">("vs")
  React.useEffect(() => {
    const resolve = () => {
      if (theme === "dark") return "vs-dark" as const
      if (theme === "light") return "vs" as const
      return document.documentElement.classList.contains("dark")
        ? ("vs-dark" as const)
        : ("vs" as const)
    }
    setMonacoTheme(resolve())
    if (theme !== "system") return
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => setMonacoTheme(resolve())
    media.addEventListener("change", handler)
    return () => media.removeEventListener("change", handler)
  }, [theme])

  // Sync state when extension changes
  const initialSyncedRef = React.useRef(false)
  React.useEffect(() => {
    if (!extension) return
    setSelectedVersion(extension.version || "")
    setModifiedValues(extension.values ?? "")
    setShowDiff(false)
    initialSyncedRef.current = false
  }, [extension])

  // Fetch full extension details to get current values
  const { data: extensionDetails } = useQuery({
    queryKey: ["cluster-extensions", clusterId, extension?.id],
    queryFn: () => clustersApi.getClusterExtension(clusterId, extension!.id),
    enabled: open && Boolean(extension?.id),
  })

  React.useEffect(() => {
    if (
      !extensionDetails ||
      !extension ||
      extensionDetails.id !== extension.id ||
      initialSyncedRef.current
    )
      return
    setModifiedValues(extensionDetails.values ?? "")
    initialSyncedRef.current = true
  }, [extension, extensionDetails])

  // Fetch versions for this extension
  const { data: versionsData = [], isLoading: versionsLoading } = useQuery({
    queryKey: ["extension-versions", extension?.extension_id],
    queryFn: () =>
      clustersApi.getExtensionVersions(extension!.extension_id!),
    enabled: open && Boolean(extension?.extension_id),
    staleTime: 5 * 60 * 1000,
  })

  const rawVersions: ExtensionVersionInfo[] = Array.isArray(versionsData)
    ? versionsData
    : []

  const versions = filterSelectableExtensionVersions(rawVersions, extension?.version)

  // Fetch default values for selected version (only in diff mode)
  const { data: chartValuesData, isFetching: chartValuesFetching } = useQuery({
    queryKey: [
      "extension-values",
      extension?.extension_id,
      selectedVersion,
    ],
    queryFn: () =>
      clustersApi.getExtensionValues(
        extension!.extension_id!,
        selectedVersion
      ),
    enabled: Boolean(
      showDiff &&
      open &&
      extension?.extension_id &&
      selectedVersion
    ),
    staleTime: 5 * 60 * 1000,
  })

  const originalValues: string =
    (chartValuesData as { values?: string } | undefined)?.values ?? ""

  const updateMutation = useMutation({
    mutationFn: (data: UpgradeExtensionRequest) =>
      clustersApi.upgradeExtension(clusterId, extension!.id, data),
    onSuccess: () => {
      toast.success(isRetryingFailedUpdate ? "Extension update retried" : "Extension updated", {
        description: `${extension?.name || extension?.release_name} ${isRetryingFailedUpdate ? "update has been queued again" : "has been updated"}.`,
      })
      queryClient.invalidateQueries({ queryKey: ["cluster-extensions", clusterId] })
      onOpenChange(false)
    },
    onError: (error: unknown) => {
      const msg =
        error &&
          typeof error === "object" &&
          "response" in error
          ? (
            error as {
              response?: { data?: { error?: string } }
            }
          ).response?.data?.error
          : null
      toast.error("Failed to update extension", {
        description:
          msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleSave = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!extension) return
    const data: UpgradeExtensionRequest = {
      version: selectedVersion || undefined,
      values: modifiedValues.trim() || undefined,
    }
    updateMutation.mutate(data)
  }

  const handleDiffMount = React.useCallback(
    (diffEditor: MonacoDiffEditor) => {
      const modified = diffEditor.getModifiedEditor()
      const model = modified.getModel()
      if (!model) return
      const disposable = model.onDidChangeContent(() => {
        setModifiedValues(modified.getValue())
      })
      return () => disposable.dispose()
    },
    []
  )

  if (!extension) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] flex-col gap-0 overflow-hidden p-0 sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
        <form
          onSubmit={handleSave}
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <DialogHeader className="shrink-0 px-6 pt-6">
            <DialogTitle>{isRetryingFailedUpdate ? "Retry Update Extension" : "Update Extension"}</DialogTitle>
            <DialogDescription>
              {isRetryingFailedUpdate ? "Retry updating" : "Update"} <span className="font-medium">{extension.name || extension.release_name}</span>{" "}
              using release <span className="font-mono text-xs">{extension.release_name}</span> in namespace <span className="font-mono text-xs">{extension.namespace}</span>.
              Edit values below and save to apply.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            <div className="flex flex-col gap-4 overflow-auto">
              <Field>
                <FieldLabel>Chart version</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={selectedVersion}
                    onValueChange={(v) => v && setSelectedVersion(v)}
                    disabled={versionsLoading}
                  >
                    <ComboboxInput placeholder={versionsLoading ? "Loading versions..." : "Select version"} />
                    <ComboboxContent>
                      <ComboboxList>
                        {versions.map((v) => (
                          <ComboboxItem key={v.version} value={v.version}>
                            {v.version}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>
              {showDiff && (
                <p className="text-xs text-muted-foreground">
                  Left: default values for selected version. Right: current
                  config (editable).
                </p>
              )}
            </div>

            <Field className="flex min-h-0 flex-1 flex-col">
              <FieldLabel className="shrink-0 flex items-center justify-between">
                <span>
                  {showDiff
                    ? "Diff: Default (left) vs current config (right)"
                    : "Values (YAML)"}
                  {showDiff && chartValuesFetching && (
                    <span className="ml-2 inline-flex items-center gap-1 text-muted-foreground">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Loading defaults...
                    </span>
                  )}
                </span>
                <Button
                  type="button"
                  variant={showDiff ? "secondary" : "outline"}
                  size="sm"
                  className="shrink-0 h-6 px-2 text-xs"
                  onClick={() => setShowDiff((v) => !v)}
                  disabled={!selectedVersion || !extension.extension_id}
                >
                  <GitCompare className="h-3 w-3" />
                  {showDiff ? "Hide diff" : "Compare with default values"}
                </Button>
              </FieldLabel>
              <FieldContent className="min-h-0 flex-1 overflow-hidden rounded-md border border-input">
                <div className="h-full min-h-75 w-full">
                  {showDiff ? (
                    <DiffEditor
                      height="100%"
                      language="yaml"
                      original={originalValues}
                      modified={modifiedValues}
                      theme={monacoTheme}
                      onMount={handleDiffMount}
                      loading={chartValuesFetching ? "Loading..." : undefined}
                      options={{
                        readOnly: false,
                        renderSideBySide: true,
                        enableSplitViewResizing: true,
                        minimap: { enabled: false },
                        fontSize: 12,
                        wordWrap: "on",
                        renderOverviewRuler: false,
                      }}
                    />
                  ) : (
                    <Editor
                      height="100%"
                      language="yaml"
                      theme={monacoTheme}
                      value={modifiedValues}
                      onChange={(v) => setModifiedValues(v ?? "")}
                      options={{
                        minimap: { enabled: false },
                        scrollBeyondLastLine: false,
                        fontSize: 12,
                        wordWrap: "on",
                        padding: { top: 8 },
                        tabSize: 2,
                      }}
                    />
                  )}
                </div>
              </FieldContent>
            </Field>
          </div>

          <DialogFooter className="shrink-0 border-t px-6 py-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="animate-spin" />
                  {isRetryingFailedUpdate ? "Retrying..." : "Updating..."}
                </>
              ) : (
                isRetryingFailedUpdate ? "Retry Update" : "Update"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
