import { appGroupsApi, type AppGroup } from '@/api/app-groups'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

interface Props {
  group: AppGroup
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function EditAppGroupDialog({ group, open, onOpenChange, onSuccess }: Props) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(group.name)
  const [description, setDescription] = useState(group.description)

  const mutation = useMutation({
    mutationFn: () => appGroupsApi.update(group.id, { name, description }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-groups', group.env_id] })
      toast.success('Group updated')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: () => toast.error('Failed to update group'),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit App Group</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-group-name">Name</Label>
            <Input id="edit-group-name" value={name} onChange={e => setName(e.target.value)} placeholder="Group name" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-group-description">Description</Label>
            <Textarea id="edit-group-description" value={description} onChange={e => setDescription(e.target.value)} placeholder="Optional description" className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={!name.trim() || mutation.isPending}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
