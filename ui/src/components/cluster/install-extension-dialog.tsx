import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type HelmChartInfo,
  type HelmChartVersionInfo,
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

interface InstallExtensionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  chart?: HelmChartInfo | null
  repositoryName?: string
}

export function InstallExtensionDialog({
  open,
  onOpenChange,
  clusterId,
  chart,
  repositoryName,
}: InstallExtensionDialogProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [releaseName, setReleaseName] = React.useState("")
  const [releaseNamespace, setReleaseNamespace] = React.useState("default")
  const [selectedVersion, setSelectedVersion] = React.useState("")
  const [values, setValues] = React.useState("")

  // Resolved theme for Monaco: match platform theme (ThemeProvider sets "light"/"dark" on document)
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

  // Reset form when chart changes
  React.useEffect(() => {
    if (chart) {
      setReleaseName(chart.name)
      setReleaseNamespace("default")
      setSelectedVersion(chart.versions?.[0]?.version || "")
      setValues("")
    }
  }, [chart])

  // Fetch chart default values when version (and repo + chart) are set
  const {
    data: chartValuesData,
    isLoading: chartValuesLoading,
    isFetching: chartValuesFetching,
    error: chartValuesError,
    isSuccess: chartValuesSuccess,
  } = useQuery({
    queryKey: [
      "chart-values",
      clusterId,
      repositoryName ?? "",
      chart?.name ?? "",
      selectedVersion,
    ],
    queryFn: async () => {
      const res = await clustersApi.getChartValues(
        clusterId,
        repositoryName!,
        chart!.name,
        selectedVersion
      )
      return res
    },
    enabled:
      Boolean(open && repositoryName && chart?.name && selectedVersion) &&
      Boolean(chart?.versions?.some((v) => v.version === selectedVersion)),
    staleTime: 5 * 60 * 1000,
  })

  // Populate values when chart values response arrives (API returns { values: string })
  React.useEffect(() => {
    if (!chartValuesSuccess || chartValuesData == null) return
    const raw = chartValuesData as { values?: string }
    if (typeof raw.values === "string") {
      setValues(raw.values)
    }
  }, [chartValuesSuccess, chartValuesData])

  const installMutation = useMutation({
    mutationFn: (data: InstallExtensionRequest) =>
      clustersApi.installExtension(clusterId, data),
    onSuccess: () => {
      toast.success("Extension installed", {
        description: `${chart?.name} is being installed to the cluster.`,
      })
      queryClient.invalidateQueries({
        queryKey: ["extensions", clusterId],
      })
      onOpenChange(false)
    },
    onError: (error: unknown) => {
      const msg =
        error && typeof error === "object" && "response" in error
          ? (error as { response?: { data?: { error?: string }; message?: string } })
              .response?.data?.error
          : null
      toast.error("Failed to install extension", {
        description: msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!chart) return

    const data: InstallExtensionRequest = {
      name: releaseName,
      chart_name: chart.name,
      chart_version: selectedVersion,
      repository: repositoryName,
      release_namespace: releaseNamespace,
      create_namespace: true,
    }
    if (values.trim()) {
      data.values = values
    }
    installMutation.mutate(data)
  }

  const versions: HelmChartVersionInfo[] = chart?.versions || []

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
              Install <span className="font-medium">{chart?.name}</span> to your
              cluster.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            {/* Left: form fields, narrow width */}
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
              </Field>

              <Field>
                <FieldLabel htmlFor="release-version">Version *</FieldLabel>
                <FieldContent>
                    <Select
                      value={selectedVersion}
                      onValueChange={(v) => setSelectedVersion(v ?? "")}
                    >
                    <SelectTrigger id="release-version" className="w-full">
                      <SelectValue placeholder="Latest" />
                    </SelectTrigger>
                    <SelectContent>
                      {versions.map((v) => (
                        <SelectItem key={v.version} value={v.version}>
                          {v.version}
                          {v.app_version
                            ? ` (app: ${v.app_version})`
                            : ""}
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
                    onChange={(e) =>
                      setReleaseNamespace(e.target.value)
                    }
                    required
                  />
                </FieldContent>
              </Field>
            </div>

            {/* Right: Values (YAML) editor, takes remaining width and full height */}
            <Field className="flex min-h-0 flex-1 flex-col">
              <FieldLabel htmlFor="values" className="shrink-0">
                Values (YAML){" "}
                <span className="font-normal text-muted-foreground">
                  (optional)
                </span>
                {chartValuesFetching && (
                  <span className="ml-2 inline-flex items-center gap-1 text-muted-foreground">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading defaults...
                  </span>
                )}
                {chartValuesError && (
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
                    loading={chartValuesLoading ? "Loading..." : undefined}
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
                chartValuesLoading ||
                chartValuesFetching
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
