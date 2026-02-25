import { DiffEditor, Editor } from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { GitCompare, Loader2 } from "lucide-react"
import type { editor } from "monaco-editor"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ExtensionVersionInfo,
  type InstalledExtension,
  type UpdateExtensionRequest,
} from "@/api/clusters"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface UpdateExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  extension: InstalledExtension | null
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
    setSelectedVersion(extension.chart_version || "")
    setModifiedValues(extension.values ?? "")
    setShowDiff(false)
    initialSyncedRef.current = false
  }, [extension])

  // Fetch full extension details to get current values
  const { data: extensionDetails } = useQuery({
    queryKey: ["extensions", clusterId, extension?.name],
    queryFn: () => clustersApi.getExtension(clusterId, extension!.name),
    enabled: open && Boolean(extension?.name),
  })

  React.useEffect(() => {
    if (
      !extensionDetails ||
      !extension ||
      extensionDetails.name !== extension.name ||
      initialSyncedRef.current
    )
      return
    setModifiedValues(extensionDetails.values ?? "")
    initialSyncedRef.current = true
  }, [extension, extensionDetails])

  // Fetch versions for this extension's catalog item
  const { data: versionsData = [] } = useQuery({
    queryKey: ["extension-versions", extension?.catalog_item_id],
    queryFn: () =>
      clustersApi.getExtensionVersions(extension!.catalog_item_id!),
    enabled: open && Boolean(extension?.catalog_item_id),
    staleTime: 5 * 60 * 1000,
  })

  const rawVersions: ExtensionVersionInfo[] = Array.isArray(versionsData)
    ? versionsData
    : []

  // Ensure current version is always in the list
  const versions: ExtensionVersionInfo[] =
    rawVersions.length > 0
      ? rawVersions
      : extension?.chart_version
        ? [{ version: extension.chart_version }]
        : []

  // Fetch default values for selected version (only in diff mode)
  const { data: chartValuesData, isFetching: chartValuesFetching } = useQuery({
    queryKey: [
      "extension-values",
      extension?.catalog_item_id,
      selectedVersion,
    ],
    queryFn: () =>
      clustersApi.getExtensionValues(
        extension!.catalog_item_id!,
        selectedVersion
      ),
    enabled: Boolean(
      showDiff &&
        open &&
        extension?.catalog_item_id &&
        selectedVersion
    ),
    staleTime: 5 * 60 * 1000,
  })

  const originalValues: string =
    (chartValuesData as { values?: string } | undefined)?.values ?? ""

  const updateMutation = useMutation({
    mutationFn: (data: UpdateExtensionRequest) =>
      clustersApi.updateExtension(clusterId, extension!.name, data),
    onSuccess: () => {
      toast.success("Extension updated", {
        description: `${extension?.name} has been updated.`,
      })
      queryClient.invalidateQueries({ queryKey: ["extensions", clusterId] })
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

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    if (!extension) return
    const data: UpdateExtensionRequest = {
      chart_version: selectedVersion || undefined,
      values: modifiedValues.trim() || undefined,
    }
    updateMutation.mutate(data)
  }

  const handleDiffMount = React.useCallback(
    (diffEditor: editor.IStandaloneDiffEditor) => {
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
            <div className="flex items-start justify-between gap-4">
              <div>
                <DialogTitle>Update Extension</DialogTitle>
                <DialogDescription>
                  Update <span className="font-medium">{extension.name}</span>{" "}
                  —{" "}
                  <span className="font-mono text-xs">{extension.oci_url}</span>
                  . Edit values below and save to apply.
                </DialogDescription>
              </div>
              <Button
                type="button"
                variant={showDiff ? "secondary" : "outline"}
                size="sm"
                className="shrink-0"
                onClick={() => setShowDiff((v) => !v)}
                disabled={!selectedVersion || !extension.catalog_item_id}
              >
                <GitCompare className="h-3.5 w-3.5" />
                {showDiff ? "Hide diff" : "Compare with default Values"}
              </Button>
            </div>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            <div className="flex flex-col gap-4 overflow-auto">
              <Field>
                <FieldLabel>Chart version</FieldLabel>
                <FieldContent>
                  <Select
                    value={selectedVersion}
                    onValueChange={(v) => setSelectedVersion(v ?? "")}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select version" />
                    </SelectTrigger>
                    <SelectContent>
                      {versions.map((v) => (
                        <SelectItem key={v.version} value={v.version}>
                          {v.version}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
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
              <FieldLabel className="shrink-0">
                {showDiff
                  ? "Diff: Default (left) vs current config (right)"
                  : "Values (YAML)"}
                {showDiff && chartValuesFetching && (
                  <span className="ml-2 inline-flex items-center gap-1 text-muted-foreground">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading defaults...
                  </span>
                )}
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
                  Updating...
                </>
              ) : (
                "Update"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
