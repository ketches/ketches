import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
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
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Item, ItemContent, ItemTitle } from "@/components/ui/item"
import { getErrorMessage } from "@/lib/utils"
import Editor from "@monaco-editor/react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

export interface ConfigFileSpec {
  id?: string
  slug: string
  mount_path: string
  file_mode: string
  content: string
}

interface ConfigFileEditorProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  app: App
  configFile?: ConfigFileSpec | null
  onSuccess?: () => void
}

const FILE_MODE_OPTIONS = [
  { value: "0644", label: "0644 (rw-r--r--)" },
  { value: "0755", label: "0755 (rwxr-xr-x)" },
  { value: "0600", label: "0600 (rw-------)" },
  { value: "0400", label: "0400 (r--------)" },
  { value: "0777", label: "0777 (rwxrwxrwx)" },
]

export function ConfigFileEditor({
  open,
  onOpenChange,
  app,
  configFile,
  onSuccess,
}: ConfigFileEditorProps) {
  const queryClient = useQueryClient()
  const { theme } = useTheme()

  const [formData, setFormData] = React.useState<ConfigFileSpec>({
    slug: "",
    mount_path: "",
    file_mode: "0644",
    content: "",
  })

  const [errors, setErrors] = React.useState<Record<string, string>>({})

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

  // Reset form when dialog opens/closes or configFile changes
  React.useEffect(() => {
    if (open) {
      if (configFile) {
        setFormData({
          id: configFile.id,
          slug: configFile.slug || "",
          mount_path: configFile.mount_path || "",
          file_mode: configFile.file_mode || "0644",
          content: configFile.content || "",
        })
      } else {
        setFormData({
          slug: "",
          mount_path: "",
          file_mode: "0644",
          content: "",
        })
      }
      setErrors({})
    }
  }, [open, configFile])

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {}

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    if (!formData.mount_path.trim()) {
      newErrors.mount_path = "Mount path is required"
    } else if (!formData.mount_path.startsWith("/")) {
      newErrors.mount_path = "Mount path must start with /"
    }

    if (!formData.content.trim()) {
      newErrors.content = "Content is required"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const saveMutation = useMutation({
    mutationFn: async (data: ConfigFileSpec) => {
      if (data.id) {
        return await appsApi.updateConfigFile(data.id, data)
      } else {
        return await appsApi.addConfigFile(app.id, data)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-config-files", app.id] })
      toast.success(
        configFile ? "Config file updated successfully" : "Config file added successfully"
      )
      onSuccess?.()
    },
    onError: (error: unknown) => {
      toast.error("Failed to save config file", {
        description: getErrorMessage(error, "Unknown error"),
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateForm()) return

    saveMutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] flex-col gap-0 overflow-hidden p-0 sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
        <form
          onSubmit={handleSubmit}
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <DialogHeader className="shrink-0 px-6 pt-6">
            <DialogTitle>
              {configFile ? "Edit Config File" : "Add Config File"}
            </DialogTitle>
            <DialogDescription>
              Mount configuration files into your application instances
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden px-6 py-4 lg:grid-cols-[minmax(0,280px)_1fr]">
            {/* Left: form fields */}
            <div className="flex flex-col gap-4 overflow-auto">
              <Field>
                <FieldLabel htmlFor="slug">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="slug"
                    value={formData.slug}
                    onChange={(e) =>
                      setFormData((prev) => ({ ...prev, slug: e.target.value }))
                    }
                    placeholder="config.yaml"
                    required
                  />
                </FieldContent>
                {errors.slug && <FieldError>{errors.slug}</FieldError>}
              </Field>

              <Field>
                <FieldLabel htmlFor="mount-path">Mount Path *</FieldLabel>
                <FieldContent>
                  <Input
                    id="mount-path"
                    value={formData.mount_path}
                    onChange={(e) =>
                      setFormData((prev) => ({ ...prev, mount_path: e.target.value }))
                    }
                    placeholder="/etc/config/config.yaml"
                    required
                  />
                </FieldContent>
                {errors.mount_path && <FieldError>{errors.mount_path}</FieldError>}
              </Field>

              <Field>
                <FieldLabel htmlFor="file-mode">File Permissions</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.file_mode}
                    onValueChange={(value: string | null) =>
                      setFormData((prev) => ({ ...prev, file_mode: value ?? "0644" }))
                    }
                    itemToStringLabel={(v) => FILE_MODE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
                  >
                    <ComboboxInput />
                    <ComboboxContent>
                      <ComboboxList>
                        {FILE_MODE_OPTIONS.map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            <Item size="xs" className="p-0">
                              <ItemContent>
                                <ItemTitle>{option.label}</ItemTitle>
                              </ItemContent>
                            </Item>
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>
            </div>

            {/* Right: Content editor */}
            <Field className="flex min-h-0 flex-1 flex-col">
              <FieldLabel htmlFor="content" className="shrink-0">
                Content *
              </FieldLabel>
              <FieldContent className="flex min-h-0 flex-1">
                <div className="h-full w-full overflow-hidden rounded-md border">
                  <Editor
                    height="100%"
                    defaultLanguage="yaml"
                    theme={monacoTheme}
                    value={formData.content}
                    onChange={(value) =>
                      setFormData((prev) => ({ ...prev, content: value || "" }))
                    }
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: "on",
                      scrollBeyondLastLine: false,
                      wordWrap: "on",
                      automaticLayout: true,
                    }}
                  />
                </div>
              </FieldContent>
              {errors.content && (
                <FieldError className="mt-2">{errors.content}</FieldError>
              )}
            </Field>
          </div>

          <DialogFooter className="shrink-0 px-6 pb-6">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={saveMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={saveMutation.isPending}>
              {saveMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              {configFile ? "Update" : "Add"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
