import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ClusterExtension,
  type ExtensionVersionInfo,
} from "@/api/clusters"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
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

interface RetryInstallExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  extension: ClusterExtension | null
}

export function RetryInstallExtensionDialog({
  open,
  onOpenChange,
  clusterId,
  extension,
}: RetryInstallExtensionDialogProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [name, setName] = React.useState("")
  const [selectedVersion, setSelectedVersion] = React.useState("")
  const [values, setValues] = React.useState("")
  const [monacoTheme, setMonacoTheme] = React.useState<"vs" | "vs-dark">("vs")

  React.useEffect(() => {
    const resolve = () => {
      if (theme === "dark") return "vs-dark" as const
      if (theme === "light") return "vs" as const
      return document.documentElement.classList.contains("dark") ? "vs-dark" as const : "vs" as const
    }
    setMonacoTheme(resolve())
    if (theme !== "system") return
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => setMonacoTheme(resolve())
    media.addEventListener("change", handler)
    return () => media.removeEventListener("change", handler)
  }, [theme])

  const { data: extensionDetails } = useQuery({
    queryKey: ["cluster-extensions", clusterId, extension?.id],
    queryFn: () => clustersApi.getClusterExtension(clusterId, extension!.id),
    enabled: open && Boolean(extension?.id),
  })

  const { data: versionsData = [], isLoading: versionsLoading } = useQuery({
    queryKey: ["extension-versions", extension?.extension_id],
    queryFn: () => clustersApi.getExtensionVersions(extension!.extension_id),
    enabled: open && Boolean(extension?.extension_id),
    staleTime: 5 * 60 * 1000,
  })

  const versions: ExtensionVersionInfo[] = Array.isArray(versionsData)
    ? versionsData
    : []
  const selectableVersions = filterSelectableExtensionVersions(versions, selectedVersion || extension?.version)

  React.useEffect(() => {
    if (!extension) return
    setName(extension.name || extension.release_name)
    setSelectedVersion(extension.version || "")
    setValues(extension.values ?? "")
  }, [extension])

  React.useEffect(() => {
    if (!extensionDetails || !extension || extensionDetails.id !== extension.id) return
    setName(extensionDetails.name || extensionDetails.release_name)
    setSelectedVersion(extensionDetails.version || "")
    setValues(extensionDetails.values ?? "")
  }, [extension, extensionDetails])

  React.useEffect(() => {
    if (selectableVersions.length > 0 && !selectedVersion) {
      setSelectedVersion(selectableVersions[0].version)
    }
  }, [selectableVersions, selectedVersion])

  const retryMutation = useMutation({
    mutationFn: () =>
      clustersApi.retryExtension(clusterId, extension!.id, {
        name: name.trim(),
        version: selectedVersion || undefined,
        values,
      }),
    onSuccess: () => {
      toast.success("Retry started", {
        description: `${extension?.name || extension?.release_name} install has been queued again.`,
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
      toast.error("Failed to retry extension install", {
        description: msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!extension || !name.trim()) return
    retryMutation.mutate()
  }

  if (!extension) return null

  const releaseName = extensionDetails?.release_name ?? extension.release_name
  const namespace = extensionDetails?.namespace ?? extension.namespace
  const createNamespace = extensionDetails?.create_namespace ?? extension.create_namespace ?? false

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] flex-col gap-0 overflow-hidden p-0 sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
        <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <DialogHeader className="shrink-0 px-6 pt-6">
            <DialogTitle>Retry Install Extension</DialogTitle>
            <DialogDescription>
              Retry installing <span className="font-medium">{extension.name || extension.release_name}</span>.
              You can adjust the display name, chart version, and values before retrying.
              Other installation fields are shown read-only.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,320px)_1fr]">
            <div className="space-y-4 overflow-auto">
              <Field>
                <FieldLabel htmlFor="retry-install-name">Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="retry-install-name"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="retry-install-release-name">Release Name</FieldLabel>
                <FieldContent>
                  <Input id="retry-install-release-name" value={releaseName} disabled />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="retry-install-version">Version</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={selectedVersion}
                    onValueChange={(value) => setSelectedVersion(value ?? "")}
                    itemToStringLabel={(value) => selectableVersions.find((item) => item.version === value)?.version ?? value ?? ""}
                    disabled={versionsLoading}
                  >
                    <ComboboxInput
                      placeholder={versionsLoading ? "Loading versions..." : "Select version"}
                    />
                    <ComboboxContent>
                      <ComboboxList>
                        <ComboboxEmpty>No versions found.</ComboboxEmpty>
                        {selectableVersions.map((item) => (
                          <ComboboxItem key={item.version} value={item.version}>
                            {item.version}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="retry-install-namespace">Namespace</FieldLabel>
                <FieldContent>
                  <Input id="retry-install-namespace" value={namespace} disabled />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="retry-install-create-namespace">Create Namespace</FieldLabel>
                <FieldContent>
                  <div className="flex min-h-9 items-center rounded-md border border-input px-3">
                    <Checkbox id="retry-install-create-namespace" checked={createNamespace} disabled />
                    <label htmlFor="retry-install-create-namespace" className="ml-2 text-sm text-muted-foreground">
                      {createNamespace ? "Enabled" : "Disabled"}
                    </label>
                  </div>
                </FieldContent>
              </Field>
            </div>

            <Field className="flex min-h-0 flex-1 flex-col">
              <FieldLabel htmlFor="retry-install-values" className="shrink-0">
                Values (YAML)
              </FieldLabel>
              <FieldContent className="min-h-0 flex-1 overflow-hidden rounded-md border border-input">
                <div className="h-full min-h-75 w-full">
                  <Editor
                    height="100%"
                    defaultLanguage="yaml"
                    language="yaml"
                    theme={monacoTheme}
                    value={values}
                    onChange={(value) => setValues(value ?? "")}
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
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={retryMutation.isPending || !name.trim()}>
              {retryMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Retry Install
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
