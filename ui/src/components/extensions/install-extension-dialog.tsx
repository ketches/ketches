import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ExtensionCatalogItem,
  type ExtensionVersionInfo,
  type InstallExtensionRequest,
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
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface InstallExtensionToClusterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  // When provided, the cluster selector is pre-filled and locked
  preselectedClusterId?: string
  catalogItem?: ExtensionCatalogItem | null
}

export function InstallExtensionToClusterDialog({
  open,
  onOpenChange,
  preselectedClusterId,
  catalogItem,
}: InstallExtensionToClusterDialogProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [selectedClusterId, setSelectedClusterId] = React.useState(
    preselectedClusterId ?? ""
  )
  const [releaseName, setReleaseName] = React.useState("")
  const [releaseNamespace, setReleaseNamespace] = React.useState("default")
  const [selectedVersion, setSelectedVersion] = React.useState("")
  const [values, setValues] = React.useState("")

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

  // Sync preselectedClusterId when it changes
  React.useEffect(() => {
    if (preselectedClusterId) setSelectedClusterId(preselectedClusterId)
  }, [preselectedClusterId])

  // Reset form when catalog item or dialog open state changes
  React.useEffect(() => {
    if (catalogItem) {
      setReleaseName(catalogItem.name)
      setReleaseNamespace("ketches-extensions")
      setSelectedVersion("")
      setValues("")
    }
  }, [catalogItem])

  // Reset cluster selection when dialog closes (unless locked)
  React.useEffect(() => {
    if (!open && !preselectedClusterId) {
      setSelectedClusterId("")
    }
  }, [open, preselectedClusterId])

  // Fetch all clusters for the cluster selector
  const { data: clusters = [], isLoading: clustersLoading } = useQuery({
    queryKey: ["clusters-simple"],
    queryFn: () => clustersApi.listSimple(),
    enabled: open,
  })

  // Fetch available versions for this catalog item
  const { data: versionsData = [], isLoading: versionsLoading } = useQuery({
    queryKey: ["extension-versions", catalogItem?.id],
    queryFn: () => clustersApi.getExtensionVersions(catalogItem!.id),
    enabled: open && Boolean(catalogItem?.id),
    staleTime: 5 * 60 * 1000,
  })

  const versions: ExtensionVersionInfo[] = Array.isArray(versionsData)
    ? versionsData
    : []

  // Default to first version when versions load
  React.useEffect(() => {
    if (versions.length > 0 && !selectedVersion) {
      setSelectedVersion(versions[0].version)
    }
  }, [versions, selectedVersion])

  // Fetch default values for the selected version
  const {
    data: valuesData,
    isLoading: valuesLoading,
    isFetching: valuesFetching,
    error: valuesError,
    isSuccess: valuesSuccess,
  } = useQuery({
    queryKey: ["extension-values", catalogItem?.id, selectedVersion],
    queryFn: () =>
      clustersApi.getExtensionValues(catalogItem!.id, selectedVersion),
    enabled: open && Boolean(catalogItem?.id && selectedVersion),
    staleTime: 5 * 60 * 1000,
  })

  // Populate editor when values arrive
  React.useEffect(() => {
    if (!valuesSuccess || valuesData == null) return
    const raw = valuesData as { values?: string }
    if (typeof raw.values === "string") {
      setValues(raw.values)
    }
  }, [valuesSuccess, valuesData])

  const installMutation = useMutation({
    mutationFn: (data: InstallExtensionRequest) =>
      clustersApi.installExtension(selectedClusterId, data),
    onSuccess: () => {
      toast.success("Extension installed", {
        description: `${catalogItem?.display_name || catalogItem?.name} is being installed to the cluster.`,
      })
      queryClient.invalidateQueries({
        queryKey: ["extensions", selectedClusterId],
      })
      onOpenChange(false)
    },
    onError: (error: unknown) => {
      const msg =
        error && typeof error === "object" && "response" in error
          ? (
            error as {
              response?: { data?: { error?: string } }
            }
          ).response?.data?.error
          : null
      toast.error("Failed to install extension", {
        description:
          msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!catalogItem || !selectedClusterId) return

    const data: InstallExtensionRequest = {
      name: releaseName,
      catalog_item_id: catalogItem.id,
      chart_version: selectedVersion || undefined,
      release_namespace: releaseNamespace,
      create_namespace: true,
    }
    if (values.trim()) {
      data.values = values
    }
    installMutation.mutate(data)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] flex-col gap-0 overflow-hidden p-0 sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
        <form
          onSubmit={handleSubmit}
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <DialogHeader className="shrink-0 px-6 pt-6">
            <DialogTitle>Install Extension</DialogTitle>
            <DialogDescription>
              Install{" "}
              <span className="font-medium">
                {catalogItem?.display_name || catalogItem?.name}
              </span>{" "}
              to a cluster.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            {/* Left: form fields */}
            <div className="flex flex-col gap-4 overflow-auto">
              {/* Cluster selector — hidden when preselected */}
              {!preselectedClusterId && (
                <Field>
                  <FieldLabel htmlFor="target-cluster">
                    Target Cluster *
                  </FieldLabel>
                  <FieldContent>
                    <Select
                      value={selectedClusterId}
                      onValueChange={(v) => setSelectedClusterId(v || "")}
                      disabled={clustersLoading}
                      required
                    >
                      <SelectTrigger id="target-cluster" className="w-full">
                        <SelectValue
                          placeholder={
                            clustersLoading
                              ? "Loading clusters..."
                              : "Select cluster"
                          }
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {clusters.map((c) => (
                          <SelectItem key={c.id} value={c.id}>
                            {c.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FieldContent>
                </Field>
              )}

              <Field>
                <FieldLabel htmlFor="release-name">Release Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="release-name"
                    value={releaseName}
                    onChange={(e) => setReleaseName(e.target.value)}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="release-version">Version *</FieldLabel>
                <FieldContent>
                  <Select
                    value={selectedVersion}
                    onValueChange={(v) => setSelectedVersion(v ?? "")}
                    disabled={versionsLoading}
                  >
                    <SelectTrigger id="release-version" className="w-full">
                      <SelectValue
                        placeholder={
                          versionsLoading
                            ? "Loading versions..."
                            : "Select version"
                        }
                      />
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

              <Field>
                <FieldLabel htmlFor="release-namespace">Namespace</FieldLabel>
                <FieldContent>
                  <Input
                    id="release-namespace"
                    value={releaseNamespace}
                    onChange={(e) => setReleaseNamespace(e.target.value)}
                    required
                  />
                </FieldContent>
              </Field>
            </div>

            {/* Right: Values YAML editor */}
            <Field className="flex min-h-0 flex-1 flex-col">
              <FieldLabel htmlFor="values" className="shrink-0">
                Values (YAML){" "}
                <span className="font-normal text-muted-foreground">
                  (optional)
                </span>
                {valuesFetching && (
                  <span className="ml-2 inline-flex items-center gap-1 text-muted-foreground">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading defaults...
                  </span>
                )}
                {valuesError && (
                  <span className="ml-2 text-destructive">
                    Failed to load defaults
                  </span>
                )}
              </FieldLabel>
              <FieldContent className="min-h-0 flex-1 overflow-hidden rounded-md border border-input">
                <div className="h-full min-h-[300px] w-full">
                  <Editor
                    height="100%"
                    defaultLanguage="yaml"
                    language="yaml"
                    theme={monacoTheme}
                    value={values}
                    onChange={(v) => setValues(v ?? "")}
                    loading={valuesLoading ? "Loading..." : undefined}
                    options={{
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                      fontSize: 12,
                      wordWrap: "on",
                      padding: { top: 8 },
                      tabSize: 2,
                    }}
                  />
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
            <Button
              type="submit"
              disabled={
                installMutation.isPending ||
                valuesLoading ||
                valuesFetching ||
                !selectedClusterId
              }
            >
              {installMutation.isPending ? (
                <>
                  <Loader2 className="animate-spin" />
                  Installing...
                </>
              ) : (
                "Install"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
