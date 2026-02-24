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

interface NodeLabelsEditorProps {
  clusterId: string
  nodeName: string
  labels: Record<string, string>
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NodeLabelsEditor({
  clusterId,
  nodeName,
  labels,
  open,
  onOpenChange,
}: NodeLabelsEditorProps) {
  const queryClient = useQueryClient()
  const [editingLabels, setEditingLabels] = React.useState<Array<{ key: string; value: string }>>([])

  React.useEffect(() => {
    if (open) {
      setEditingLabels(
        Object.entries(labels || {}).map(([key, value]) => ({ key, value }))
      )
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: (newLabels: Record<string, string>) =>
      clustersApi.updateNodeLabels(clusterId, nodeName, newLabels),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster-node", clusterId, nodeName] })
      toast.success("Labels updated successfully")
      onOpenChange(false)
    },
    onError: (error: any) => {
      toast.error("Failed to update labels", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleAdd = () => {
    setEditingLabels([...editingLabels, { key: "", value: "" }])
  }

  const handleRemove = (index: number) => {
    setEditingLabels(editingLabels.filter((_, i) => i !== index))
  }

  const handleChange = (index: number, field: "key" | "value", value: string) => {
    const newList = [...editingLabels]
    newList[index][field] = value
    setEditingLabels(newList)
  }

  const handleSave = () => {
    const labelsObj: Record<string, string> = {}
    for (const { key, value } of editingLabels) {
      if (key.trim()) {
        labelsObj[key.trim()] = value.trim()
      }
    }
    mutation.mutate(labelsObj)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Manage Labels</DialogTitle>
          <DialogDescription>
            Edit labels for node {nodeName}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {editingLabels.map((label, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                placeholder="Key"
                value={label.key}
                onChange={(e) => handleChange(index, "key", e.target.value)}
                className="flex-1 font-mono text-xs"
              />
              <span className="text-muted-foreground">:</span>
              <Input
                placeholder="Value"
                value={label.value}
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
            className="border-dashed"
          >
            <Plus />
            Add Label
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={mutation.isPending}>
              {mutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
