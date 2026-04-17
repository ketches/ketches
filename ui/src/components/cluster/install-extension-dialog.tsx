import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { InfoIcon, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type Extension,
  type ExtensionVersionInfo,
  type InstallExtensionRequest,
} from "@/api/clusters"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import { Input } from "@/components/ui/input"
import { filterSelectableExtensionVersions } from "@/lib/extension-versions"
import { sanitizeHelmReleaseName } from "@/lib/helm-release-name"

interface InstallExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  extension?: Extension | null
}

export function InstallExtensionDialog({
  open,
  onOpenChange,
  clusterId,
  extension: extension,
}: InstallExtensionDialogProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [releaseName, setReleaseName] = React.useState("")
  const [releaseNamespace, setReleaseNamespace] = React.useState("default")
  const [selectedVersion, setSelectedVersion] = React.useState("")
  const [values, setValues] = React.useState("")
  const [createNamespace, setCreateNamespace] = React.useState(false)
  const lastSuggestedNamespaceRef = React.useRef("")

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

  // Reset form when extension changes
  React.useEffect(() => {
    if (extension) {
      setReleaseName(sanitizeHelmReleaseName(extension.name))
      setReleaseNamespace("default")
      setSelectedVersion("")
      setValues("")
      setCreateNamespace(false)
      lastSuggestedNamespaceRef.current = ""
    }
  }, [extension])

  // Fetch available versions for this extension
  const { data: versionsData = [], isLoading: versionsLoading } = useQuery({
    queryKey: ["extension-versions", extension?.id],
    queryFn: () => clustersApi.getExtensionVersions(extension!.id),
    enabled: open && Boolean(extension?.id),
    staleTime: 5 * 60 * 1000,
  })

  const versions: ExtensionVersionInfo[] = Array.isArray(versionsData)
    ? versionsData
    : []
  const selectableVersions = filterSelectableExtensionVersions(versions, selectedVersion)

  // Default to first version when versions load
  React.useEffect(() => {
    if (selectableVersions.length > 0 && !selectedVersion) {
      setSelectedVersion(selectableVersions[0].version)
    }
  }, [selectableVersions, selectedVersion])

  const normalizedNamespace = releaseNamespace.trim()

  const { data: namespaces = [], isSuccess: namespacesLoaded, isFetching: namespacesFetching } = useQuery({
    queryKey: ["cluster-namespaces", clusterId],
    queryFn: () => clustersApi.listNamespaces(clusterId),
    enabled: open && !!clusterId,
    staleTime: 60 * 1000,
  })

  const namespaceExists = normalizedNamespace !== "" && namespaces.includes(normalizedNamespace)
  const namespaceCreationBlocked = normalizedNamespace !== "" && namespacesLoaded && !namespaceExists && !createNamespace

  const namespaceStatus = React.useMemo(() => {
    if (!normalizedNamespace) {
      return null
    }

    if (namespacesFetching) {
      return {
        tone: "muted" as const,
        text: "Checking...",
      }
    }

    if (!namespacesLoaded) {
      return null
    }

    if (namespaceExists) {
      return {
        tone: "success" as const,
        text: "Exists in cluster",
      }
    }

    if (namespaceCreationBlocked) {
      return {
        tone: "warning" as const,
        text: "Create namespace required",
      }
    }

    return {
      tone: "warning" as const,
      text: "Will be created",
    }
  }, [namespaceCreationBlocked, namespaceExists, namespacesFetching, namespacesLoaded, normalizedNamespace])

  React.useEffect(() => {
    if (!open || !normalizedNamespace || !namespacesLoaded) {
      return
    }
    if (namespaceExists) {
      setCreateNamespace(false)
      lastSuggestedNamespaceRef.current = normalizedNamespace
      return
    }
    if (lastSuggestedNamespaceRef.current !== normalizedNamespace) {
      setCreateNamespace(true)
      lastSuggestedNamespaceRef.current = normalizedNamespace
    }
  }, [namespaceExists, namespacesLoaded, normalizedNamespace, open])

  // Fetch default values for the selected version
  const {
    data: valuesData,
    isLoading: valuesLoading,
    isFetching: valuesFetching,
    error: valuesError,
    isSuccess: valuesSuccess,
  } = useQuery({
    queryKey: ["extension-values", extension?.id, selectedVersion],
    queryFn: () =>
      clustersApi.getExtensionValues(extension!.id, selectedVersion),
    enabled: open && Boolean(extension?.id && selectedVersion),
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
      clustersApi.installExtension(clusterId, data),
    onSuccess: () => {
      toast.success("Extension installed", {
        description: `${extension?.display_name || extension?.name} is being installed to the cluster.`,
      })
      queryClient.invalidateQueries({ queryKey: ["cluster-extensions", clusterId] })
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

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!extension) return

    const data: InstallExtensionRequest = {
      release_name: releaseName,
      extension_id: extension.id,
      version: selectedVersion || undefined,
      namespace: releaseNamespace,
      create_namespace: namespaceExists ? false : createNamespace,
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
                {extension?.display_name || extension?.name}
              </span>{" "}
              to your cluster.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            {/* Left: form fields */}
            <div className="flex flex-col gap-4 overflow-auto">
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
                <p className="text-muted-foreground text-xs">Use lowercase letters, numbers, hyphens, or dots, up to 53 characters.</p>
              </Field>

              <Field>
                <FieldLabel htmlFor="release-version">Version *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={selectedVersion}
                    onValueChange={(v) => v && setSelectedVersion(v)}
                    disabled={versionsLoading}
                  >
                    <ComboboxInput placeholder={versionsLoading ? "Loading versions..." : "Select version"} />
                    <ComboboxContent>
                      <ComboboxList>
                        {selectableVersions.map((v) => (
                          <ComboboxItem key={v.version} value={v.version}>
                            {v.version}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="release-namespace" className="flex items-center justify-between gap-3">
                  <span>Namespace</span>
                  {namespaceStatus && (
                    <span
                      className={
                        namespaceStatus.tone === "success"
                          ? "text-xs text-emerald-600"
                          : namespaceStatus.tone === "warning"
                            ? "text-xs text-amber-700"
                            : "text-xs text-muted-foreground"
                      }
                    >
                      {namespaceStatus.text}
                    </span>
                  )}
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="release-namespace"
                    value={releaseNamespace}
                    onChange={(e) => setReleaseNamespace(e.target.value)}
                    required
                  />
                </FieldContent>
                {normalizedNamespace && namespacesLoaded && !namespaceExists && (
                  <div className="space-y-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-amber-700">
                    <div className="flex items-start gap-2">
                      <InfoIcon className="mt-0.5 h-4 w-4 shrink-0" />
                      <p className="text-xs">Namespace "{normalizedNamespace}" does not exist in the cluster and needs to be created before installation.</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="create-namespace"
                        checked={createNamespace}
                        onCheckedChange={(checked) => setCreateNamespace(checked === true)}
                      />
                      <label htmlFor="create-namespace" className="cursor-pointer text-xs">
                        Create namespace
                      </label>
                    </div>
                  </div>
                )}
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
                <div className="h-full min-h-75 w-full">
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
                installMutation.isPending || valuesLoading || valuesFetching || namespaceCreationBlocked
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
