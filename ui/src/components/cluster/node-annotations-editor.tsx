import { AxiosError } from "axios"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi } from "@/api/clusters"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"

interface NodeAnnotationsEditorProps {
  clusterId: string
  nodeName: string
  annotations: Record<string, string>
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NodeAnnotationsEditor({
  clusterId,
  nodeName,
  annotations,
  open,
  onOpenChange,
}: NodeAnnotationsEditorProps) {
  const queryClient = useQueryClient()
  const [editingAnnotations, setEditingAnnotations] = React.useState<Array<{ key: string; value: string }>>([])

  React.useEffect(() => {
    if (open) {
      setEditingAnnotations(
        Object.entries(annotations || {}).map(([key, value]) => ({ key, value }))
      )
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: (newAnnotations: Record<string, string>) =>
      clustersApi.updateNodeAnnotations(clusterId, nodeName, newAnnotations),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster-node", clusterId, nodeName] })
      toast.success("Annotations updated successfully")
      onOpenChange(false)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to update annotations", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleAdd = () => {
    setEditingAnnotations([...editingAnnotations, { key: "", value: "" }])
  }

  const handleRemove = (index: number) => {
    setEditingAnnotations(editingAnnotations.filter((_, i) => i !== index))
  }

  const handleChange = (index: number, field: "key" | "value", value: string) => {
    const newList = [...editingAnnotations]
    newList[index][field] = value
    setEditingAnnotations(newList)
  }

  const handleSave = () => {
    const annotationsObj: Record<string, string> = {}
    for (const { key, value } of editingAnnotations) {
      if (key.trim()) {
        annotationsObj[key.trim()] = value.trim()
      }
    }
    mutation.mutate(annotationsObj)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Manage Annotations</DialogTitle>
          <DialogDescription>
            Edit annotations for node {nodeName}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {editingAnnotations.map((annotation, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                placeholder="Key"
                value={annotation.key}
                onChange={(e) => handleChange(index, "key", e.target.value)}
                className="flex-1 font-mono text-xs"
              />
              <span className="text-muted-foreground">:</span>
              <Input
                placeholder="Value"
                value={annotation.value}
                onChange={(e) => handleChange(index, "value", e.target.value)}
                className="flex-1 font-mono text-xs"
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleRemove(index)}
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
              >
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
        <DialogFooter className="flex-row sm:justify-between items-center">
          <Button
            variant="outline"
            onClick={handleAdd}
          >
            <Plus />
            Add Annotation
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button size="sm" onClick={handleSave} disabled={mutation.isPending}>
              {mutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
