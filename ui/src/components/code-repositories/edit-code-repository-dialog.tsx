import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { codeRepositoriesApi, type CodeRepository, type UpdateCodeRepositoryRequest } from "@/api/code-repositories"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface EditCodeRepositoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repo: CodeRepository | null
  onSuccess?: () => void
}

export function EditCodeRepositoryDialog({ open, onOpenChange, repo, onSuccess }: EditCodeRepositoryDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<UpdateCodeRepositoryRequest>({})
  const [hasPassword, setHasPassword] = React.useState(false)

  React.useEffect(() => {
    if (repo && open) {
      setForm({
        name: repo.name,
        slug: repo.slug,
        git_repo_url: repo.git_repo_url,
        git_username: repo.git_username ?? '',
        webhook_enabled: repo.webhook_enabled,
      })
      setHasPassword(repo.has_git_password ?? false)
    }
  }, [repo, open])

  const updateMutation = useMutation({
    mutationFn: () => {
      const { slug: _slug, ...updateData } = form
      return codeRepositoriesApi.update(repo!.id, updateData)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['code-repository', repo?.id] })
      queryClient.invalidateQueries({ queryKey: ['code-repositories'] })
      toast.success('Code repository updated')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (err: unknown) => {
      const msg = err && typeof err === 'object' && 'response' in err
        ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
        : null
      toast.error(msg || 'Failed to update code repository')
    },
  })

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault()
    if (!form.name?.trim() || !form.git_repo_url?.trim()) {
      toast.error('Name and Git URL are required')
      return
    }
    if (form.slug !== undefined && !form.slug?.trim()) {
      toast.error('Slug is required')
      return
    }
    updateMutation.mutate()
  }

  if (!repo) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Code Repository</DialogTitle>
            <DialogDescription>
              Update name, URL, and credentials. Build configuration is managed in the build configs section below.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    value={form.name ?? ''}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    value={form.slug ?? ''}
                    disabled
                    className="bg-muted font-mono"
                  />
                </FieldContent>
              </Field>
            </div>
            <Field>
              <FieldLabel>Git Repository URL *</FieldLabel>
              <FieldContent>
                <Input
                  value={form.git_repo_url ?? ''}
                  onChange={(e) => setForm({ ...form, git_repo_url: e.target.value })}
                  required
                />
              </FieldContent>
            </Field>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Git Username</FieldLabel>
                <FieldContent>
                  <Input
                    value={form.git_username ?? ''}
                    onChange={(e) => setForm({ ...form, git_username: e.target.value })}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Git Password / Token</FieldLabel>
                <FieldContent>
                  <Input
                    type="password"
                    placeholder={hasPassword ? "••••••••" : "Enter password/token"}
                    value={form.git_password ?? ''}
                    onChange={(e) => setForm({ ...form, git_password: e.target.value })}
                  />
                </FieldContent>
              </Field>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="webhook-enabled-checkbox"
                checked={form.webhook_enabled ?? false}
                onCheckedChange={(v) => setForm({ ...form, webhook_enabled: v === true })}
              />
              <label htmlFor="webhook-enabled-checkbox" className="cursor-pointer">
                Webhook enabled
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog >
  )
}
