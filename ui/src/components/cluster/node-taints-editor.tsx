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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface NodeTaintsEditorProps {
  clusterId: string
  nodeName: string
  taints?: Array<{ key: string; value?: string; effect: string }>
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NodeTaintsEditor({
  clusterId,
  nodeName,
  taints,
  open,
  onOpenChange,
}: NodeTaintsEditorProps) {
  const queryClient = useQueryClient()
  const [editingTaints, setEditingTaints] = React.useState<Array<{ taint_key: string; taint_value: string; effect: string }>>([])

  React.useEffect(() => {
    if (open) {
      setEditingTaints(
        (taints || []).map((t) => ({
          taint_key: t.key,
          taint_value: t.value || "",
          effect: t.effect
        }))
      )
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: (newTaints: Array<{ taint_key: string; taint_value?: string; effect: string }>) =>
      clustersApi.updateNodeTaints(clusterId, nodeName, newTaints),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster-node", clusterId, nodeName] })
      toast.success("Taints updated successfully")
      onOpenChange(false)
    },
    onError: (error: any) => {
      toast.error("Failed to update taints", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleAdd = () => {
    setEditingTaints([...editingTaints, { taint_key: "", taint_value: "", effect: "NoSchedule" }])
  }

  const handleRemove = (index: number) => {
    setEditingTaints(editingTaints.filter((_, i) => i !== index))
  }

  const handleChange = (index: number, field: string, value: string) => {
    const newList = [...editingTaints]
    // @ts-ignore
    newList[index][field] = value
    setEditingTaints(newList)
  }

  const handleSave = () => {
    const taintsToSave = editingTaints
      .filter(t => t.taint_key.trim())
      .map(t => ({
        taint_key: t.taint_key.trim(),
        taint_value: t.taint_value.trim() || undefined,
        effect: t.effect
      }))
    mutation.mutate(taintsToSave)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-220 max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Manage Taints</DialogTitle>
          <DialogDescription>
            Edit taints for node {nodeName}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {editingTaints.map((taint, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                placeholder="Key"
                value={taint.taint_key}
                onChange={(e) => handleChange(index, "taint_key", e.target.value)}
                className="flex-1 font-mono text-xs"
              />
              <span className="text-muted-foreground">:</span>
              <Input
                placeholder="Value (optional)"
                value={taint.taint_value}
                onChange={(e) => handleChange(index, "taint_value", e.target.value)}
                className="flex-1 font-mono text-xs"
              />
              <Select
                value={taint.effect}
                onValueChange={(v) => handleChange(index, "effect", v as string)}
              >
                <SelectTrigger className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="NoSchedule">NoSchedule</SelectItem>
                  <SelectItem value="PreferNoSchedule">PreferNoSchedule</SelectItem>
                  <SelectItem value="NoExecute">NoExecute</SelectItem>
                </SelectContent>
              </Select>
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
            Add Taint
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
