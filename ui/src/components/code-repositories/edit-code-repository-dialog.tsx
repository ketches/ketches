import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Key, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { codeRepositoriesApi, type CodeRepository, type UpdateCodeRepositoryRequest } from "@/api/code-repositories"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface EditCodeRepositoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repo: CodeRepository | null
  onSuccess?: () => void
}

export function EditCodeRepositoryDialog({ open, onOpenChange, repo, onSuccess }: EditCodeRepositoryDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<UpdateCodeRepositoryRequest>({})
  const [showCredentials, setShowCredentials] = React.useState(false)
  const [isClearingPassword, setIsClearingPassword] = React.useState(false)

  React.useEffect(() => {
    if (repo && open) {
      setForm({
        name: repo.name,
        git_repo_url: repo.git_repo_url,
        git_username: repo.git_username ?? '',
        git_password: repo.git_password ?? '',
        clear_git_password: false,
      })
      setShowCredentials(Boolean(repo.has_git_password || repo.git_username || repo.git_password))
      setIsClearingPassword(false)
    }
  }, [repo, open])

  const updateMutation = useMutation({
    mutationFn: () => {
      const { ...updateData } = form
      if (isClearingPassword && !updateData.git_password) {
        updateData.clear_git_password = true
      }
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
              Update name, URL, and credentials.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="name">Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="name"
                    value={form.name ?? ''}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="slug">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="slug"
                    value={repo.slug ?? ''}
                    disabled
                    className="bg-muted font-mono"
                  />
                </FieldContent>
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="git-repo-url">Git Repository URL *</FieldLabel>
              <FieldContent>
                <InputGroup>
                  <InputGroupInput
                    id="git-repo-url"
                    value={form.git_repo_url ?? ''}
                    onChange={(e) => setForm({ ...form, git_repo_url: e.target.value })}
                    required />
                  <InputGroupAddon align="inline-end">
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant={showCredentials ? "default" : "ghost"}
                            size="icon-sm"
                            aria-label="Git credentials"
                            aria-pressed={showCredentials}
                            onClick={() => setShowCredentials((prev) => !prev)}
                            className="ml-auto"
                          />
                        }
                      >
                        <Key />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Git Credentials</p>
                      </TooltipContent>
                    </Tooltip>
                  </InputGroupAddon>
                </InputGroup>
              </FieldContent>
            </Field>
            {showCredentials && (
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel htmlFor="git-username">Git Username</FieldLabel>
                  <FieldContent>
                    <Input
                      id="git-username"
                      value={form.git_username ?? ''}
                      onChange={(e) => setForm({ ...form, git_username: e.target.value })}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel htmlFor="git-password">Git Password / Token</FieldLabel>
                  <FieldContent>
                    {repo.has_git_password && !isClearingPassword ? (
                      <div className="flex h-9 items-center justify-between rounded-md border border-input bg-transparent px-3 py-1 shadow-sm">
                        <span className="text-sm text-muted-foreground">********</span>
                        <Button type="button" variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={() => setIsClearingPassword(true)}>
                          Clear Password
                        </Button>
                      </div>
                    ) : (
                      <Input
                      id="git-password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="Enter password/token"
                      value={form.git_password ?? ''}
                      onChange={(e) => setForm({ ...form, git_password: e.target.value })}
                    />
                    )}
                  </FieldContent>
                </Field>
              </div>
            )}
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
