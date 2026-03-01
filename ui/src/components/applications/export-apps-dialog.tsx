import { appsApi } from '@/api/apps'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { AlertCircle } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { FieldContent, FieldLabel } from '../ui/field'

interface ExportAppsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  envId: string
  appIds?: string[] // selected apps
  appId?: string // single app export
  onSuccess?: () => void
}

type ExportFormat = 'kubernetes' | 'ketches' | 'helm'
type ExportScope = 'selected' | 'all'

const EXPORT_FORMAT_OPTIONS = [
  { value: 'kubernetes', label: 'Kubernetes Manifests', description: 'Raw Kubernetes YAML resources' },
  { value: 'ketches', label: 'Ketches Metadata', description: 'Ketches-native application format' },
  { value: 'helm', label: 'Helm Chart', description: 'Packaged Helm chart format' },
]


const downloadFile = (content: string, filename: string, contentType: string, isBase64 = false) => {
  let url: string

  if (isBase64) {
    // For base64 content (like zip files), we need to convert it to a blob
    const byteCharacters = atob(content)
    const byteNumbers = new Array(byteCharacters.length)
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i)
    }
    const byteArray = new Uint8Array(byteNumbers)
    const blob = new Blob([byteArray], { type: contentType })
    url = URL.createObjectURL(blob)
  } else {
    // For text content
    const blob = new Blob([content], { type: contentType })
    url = URL.createObjectURL(blob)
  }

  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export function ExportAppsDialog({
  open,
  onOpenChange,
  envId,
  appIds,
  appId,
  onSuccess,
}: ExportAppsDialogProps) {
  const [format, setFormat] = useState<ExportFormat>('kubernetes')
  const [scope, setScope] = useState<ExportScope>('selected')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Determine if we are in "batch mode" (multiple apps selected or potentially all)
  const isBatchMode = !appId

  // If appIds is empty or undefined in batch mode, default to 'all' and disable selection?
  // Or if appIds is provided, allow choosing between 'selected' and 'all'.
  const hasSelection = appIds && appIds.length > 0
  const effectiveScope = isBatchMode ? (hasSelection ? scope : 'all') : 'selected'

  const handleExport = async () => {
    setLoading(true)
    setError(null)

    try {
      let data: { yaml?: string; metadata?: string; chart?: string }

      if (appId) {
        // Single app export
        data = await appsApi.exportApps(appId, format)
      } else {
        // Batch export (env apps)
        const targetAppIds = effectiveScope === 'selected' ? appIds : undefined
        data = await appsApi.exportEnvApps(envId, format, targetAppIds)
      }

      let content = ''
      let filename = ''
      let contentType = ''
      let isBase64 = false

      const timestamp = new Date().toISOString().replace(/[:.]/g, '-')

      switch (format) {
        case 'kubernetes':
          if (data.yaml) {
            content = data.yaml
            filename = `k8s-export-${timestamp}.yaml`
            contentType = 'text/yaml'
          }
          break
        case 'ketches':
          if (data.metadata) {
            content = data.metadata
            filename = `ketches-export-${timestamp}.ketches`
            contentType = 'application/json'
          }
          break
        case 'helm':
          if (data.chart) {
            content = data.chart
            filename = `helm-chart-${timestamp}.zip`
            contentType = 'application/zip'
            isBase64 = true
          }
          break
      }

      if (!content) {
        throw new Error('No content received from export API')
      }

      downloadFile(content, filename, contentType, isBase64)

      toast.success('Export successful')
      onSuccess?.()
      onOpenChange(false)
    } catch (err: any) {
      console.error('Export failed:', err)
      setError(err.message || 'Failed to export applications')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140">
        <DialogHeader>
          <DialogTitle>Export Applications</DialogTitle>
          <DialogDescription>
            Export your application configurations in various formats.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <FieldLabel htmlFor="format" className="text-right">
            Format
          </FieldLabel>
          <FieldContent>
            <Combobox
              value={format}
              onValueChange={(v: string | null) => v && setFormat(v as ExportFormat)}
              itemToStringLabel={(v) => EXPORT_FORMAT_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
            >
              <ComboboxInput />
              <ComboboxContent>
                <ComboboxList>
                  {[
                    { value: 'kubernetes', label: 'Kubernetes Manifests', description: 'Raw Kubernetes YAML resources' },
                    { value: 'ketches', label: 'Ketches Metadata', description: 'Ketches-native application format' },
                    { value: 'helm', label: 'Helm Chart', description: 'Packaged Helm chart format' },
                  ].map((option) => (
                    <ComboboxItem key={option.value} value={option.value}>
                      <div className="flex flex-col gap-0.5">
                        <span>{option.label}</span>
                        <span className="text-muted-foreground text-[10px] leading-relaxed">{option.description}</span>
                      </div>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </FieldContent>

          {isBatchMode && hasSelection && (
            <>
              <FieldLabel htmlFor="scope" className="text-right">
                Scope
              </FieldLabel>
              <FieldContent>
                <Combobox
                  value={scope}
                  onValueChange={(v: string | null) => v && setScope(v as ExportScope)}
                  itemToStringLabel={(v) => {
                    if (v === 'selected') return `Selected Apps (${appIds?.length})`
                    if (v === 'all') return 'All Apps in Environment'
                    return v ?? ""
                  }}
                >
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {[
                        { value: 'selected', label: `Selected Apps (${appIds?.length})` },
                        { value: 'all', label: 'All Apps in Environment' },
                      ].map((option) => (
                        <ComboboxItem key={option.value} value={option.value}>
                          {option.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button onClick={handleExport} disabled={loading}>
            {loading ? 'Exporting...' : 'Export'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
