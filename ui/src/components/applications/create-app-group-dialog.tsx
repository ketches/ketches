import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { appGroupsApi } from '@/api/app-groups'

interface Props {
  projectId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function CreateAppGroupDialog({ projectId, open, onOpenChange, onSuccess }: Props) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  const mutation = useMutation({
    mutationFn: () => appGroupsApi.create(projectId, { name, description }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-groups', projectId] })
      queryClient.invalidateQueries({ queryKey: ['grouped-apps', projectId] })
      toast.success('Group created')
      setName('')
      setDescription('')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: () => toast.error('Failed to create group'),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create App Group</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="group-name">Name</Label>
            <Input id="group-name" value={name} onChange={e => setName(e.target.value)} placeholder="Group name" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-description">Description</Label>
            <Textarea id="group-description" value={description} onChange={e => setDescription(e.target.value)} placeholder="Optional description" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={!name.trim() || mutation.isPending}>Create</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
