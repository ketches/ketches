import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, Upload } from "lucide-react"
import * as React from "react"
import { toast as sonnerToast } from "sonner"
import type { AxiosError } from "axios"

import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Input } from "@/components/ui/input"

interface ImportAppsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  envId: string
  onSuccess?: () => void
}

export function ImportAppsDialog({
  open,
  onOpenChange,
  envId,
  onSuccess,
}: ImportAppsDialogProps) {
  const queryClient = useQueryClient()
  
  const [importType, setImportType] = React.useState<'dockercompose' | 'kubernetes' | 'ketches'>('dockercompose')
  const [content, setContent] = React.useState('')
  const [conflictStrategy, setConflictStrategy] = React.useState<'rename' | 'ask' | 'error'>('rename')
  const [fileName, setFileName] = React.useState('')

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const text = event.target?.result as string
      setContent(text)
      setFileName(file.name)

      // Auto-detect import type based on file extension or content
      if (file.name.endsWith('.json') || file.name.endsWith('.ketches')) {
        setImportType('ketches')
      } else if (file.name.includes('docker-compose') || file.name.includes('compose')) {
        setImportType('dockercompose')
      } else {
        // Try to detect from content
        if (text.includes('kind: Deployment') || text.includes('kind: StatefulSet')) {
          setImportType('kubernetes')
        } else if (text.includes('services:')) {
          setImportType('dockercompose')
        }
      }
    }
    reader.readAsText(file)
  }

  const mutation = useMutation({
    mutationFn: (data: { 
      type: 'dockercompose' | 'kubernetes' | 'ketches', 
      content: string, 
      conflict_strategy: 'rename' | 'ask' | 'error' 
    }) => appsApi.importApps(envId, data),
    onSuccess: (data) => {
      sonnerToast.success(`Successfully imported ${data.imported.length} applications`)
      queryClient.invalidateQueries({ queryKey: ['apps', envId] })
      onSuccess?.()
      onOpenChange(false)
      setContent('')
      setFileName('')
      setImportType('dockercompose')
      setConflictStrategy('rename')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      sonnerToast.error(err.response?.data?.error || "Failed to import applications")
    }
  })

  const handleImport = () => {
    if (!content.trim()) return
    mutation.mutate({
      type: importType,
      content,
      conflict_strategy: conflictStrategy
    })
  }

  const getPlaceholder = () => {
    switch (importType) {
      case 'dockercompose':
        return "version: '3'\nservices:\n  web:\n    image: nginx\n    ports:\n      - \"80:80\""
      case 'kubernetes':
        return "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\nspec:\n  selector:\n    matchLabels:\n      app: nginx\n  replicas: 2\n  template:\n    metadata:\n      labels:\n        app: nginx\n    spec:\n      containers:\n      - name: nginx\n        image: nginx:1.14.2\n        ports:\n        - containerPort: 80"
      case 'ketches':
        return "{\n  \"apps\": [\n    {\n      \"name\": \"my-app\",\n      \"slug\": \"my-app\",\n      \"image\": \"nginx\"\n    }\n  ]\n}"
      default:
        return ""
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Import Applications</DialogTitle>
          <DialogDescription>
            Import applications from Docker Compose, Kubernetes manifests, or Ketches metadata.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <Tabs value={importType} onValueChange={(v) => setImportType(v as 'dockercompose' | 'kubernetes' | 'ketches')} className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="dockercompose">Docker Compose</TabsTrigger>
              <TabsTrigger value="kubernetes">Kubernetes</TabsTrigger>
              <TabsTrigger value="ketches">Ketches</TabsTrigger>
            </TabsList>
          </Tabs>
          
          <div className="grid gap-2">
            <Label>Upload File</Label>
            <div className="flex items-center gap-2">
              <Input
                type="file"
                accept=".yaml,.yml,.json,.ketches"
                onChange={handleFileUpload}
                className="flex-1"
              />
              {fileName && (
                <span className="text-sm text-muted-foreground truncate max-w-[150px]">
                  {fileName}
                </span>
              )}
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">
                Or paste content below
              </span>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="content">Configuration Content</Label>
            <Textarea
              id="content"
              placeholder={getPlaceholder()}
              className="min-h-[300px] font-mono text-xs"
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="strategy">Conflict Strategy</Label>
            <Select value={conflictStrategy} onValueChange={(v) => setConflictStrategy(v as 'rename' | 'ask' | 'error')}>
              <SelectTrigger id="strategy">
                <SelectValue placeholder="Select strategy" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="rename">Auto-rename (append suffix)</SelectItem>
                <SelectItem value="ask">Ask (interactive)</SelectItem>
                <SelectItem value="error">Error (fail if exists)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {mutation.isError && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                {(mutation.error as AxiosError<{ error: string }>)?.response?.data?.error || mutation.error.message}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleImport} disabled={mutation.isPending || !content.trim()}>
            {mutation.isPending ? "Importing..." : "Import"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default ImportAppsDialog
