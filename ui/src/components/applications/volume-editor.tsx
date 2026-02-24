import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Database, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface VolumeEditorProps {
  appId: string
}

export function VolumeEditor({ appId }: VolumeEditorProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = React.useState({
    slug: "",
    mountPath: "",
    volumeType: "pvc",
    capacity: 1
  })

  const { data: volumes = [], isLoading } = useQuery({
    queryKey: ['app-volumes', appId],
    queryFn: () => appsApi.listVolumes(appId)
  })

  const addMutation = useMutation({
    mutationFn: (data: any) => appsApi.addVolume(appId, {
      slug: data.slug,
      mountPath: data.mountPath,
      volumeType: data.volumeType,
      capacity: parseInt(data.capacity)
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-volumes', appId] })
      toast.success("Volume added")
      setFormData({ slug: "", mountPath: "", volumeType: "pvc", capacity: 1 })
    },
    onError: (err: any) => {
      toast.error("Failed to add volume", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => appsApi.deleteVolume(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-volumes', appId] })
      toast.success("Volume removed")
    },
    onError: () => {
      toast.error("Failed to remove volume")
    }
  })

  if (isLoading) return <div>Loading Volumes...</div>

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Storage Volumes</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          {volumes.map((v: any) => (
            <div key={v.ID} className="flex items-center gap-2 p-2 border rounded bg-muted/30">
              <Database className="h-4 w-4 text-muted-foreground" />
              <div className="flex-1">
                <div className="font-medium">{v.Slug}</div>
                <div className="text-xs text-muted-foreground">{v.MountPath} ({v.Capacity}Gi)</div>
              </div>
              <Button variant="ghost" size="icon" onClick={() => deleteMutation.mutate(v.ID)}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          ))}
        </div>

        <div className="grid grid-cols-2 gap-2 pt-4 border-t">
          <Input
            placeholder="Name (slug)"
            value={formData.slug}
            onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
          />
          <Input
            placeholder="Mount Path"
            value={formData.mountPath}
            onChange={(e) => setFormData({ ...formData, mountPath: e.target.value })}
          />
          <Select
            value={formData.volumeType}
            onValueChange={(v) => setFormData({ ...formData, volumeType: v || "pvc" })}
            items={[
              { value: "pvc", label: "PersistentVolumeClaim (PVC)" },
              { value: "emptyDir", label: "EmptyDir (Ephemeral)" },
            ]}
          >
            <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="pvc">PersistentVolumeClaim (PVC)</SelectItem>
              <SelectItem value="emptyDir">EmptyDir (Ephemeral)</SelectItem>
            </SelectContent>
          </Select>
          <div className="flex gap-2">
            <Input
              type="number"
              placeholder="Gi"
              value={formData.capacity}
              onChange={(e) => setFormData({ ...formData, capacity: parseInt(e.target.value) || 0 })}
            />
            <Button onClick={() => addMutation.mutate(formData)} disabled={!formData.slug || !formData.mountPath}>
              Add
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
