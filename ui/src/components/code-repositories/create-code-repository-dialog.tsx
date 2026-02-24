import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { CodeRepository } from "@/api/code-repositories"
import { codeRepositoriesApi, type CreateCodeRepositoryRequest } from "@/api/code-repositories"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface CreateCodeRepositoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: (repo: CodeRepository) => void
}

function parseRepoUrl(gitRepoUrl: string): { name: string; slug: string } {
  const u = gitRepoUrl.trim()
  if (!u) return { name: '', slug: '' }
  try {
    let path = ''
    if (u.includes('@') && u.includes(':') && !u.startsWith('http') && !u.startsWith('ssh://')) {
      // Handle git@github.com:user/repo.git format
      path = u.split(':').pop() || ''
    } else {
      const url = new URL(u.startsWith('http') || u.startsWith('ssh://') ? u : `https://${u}`)
      path = url.pathname
    }

    path = path.replace(/^\/+/, '').replace(/\/+$/, '')
    const segment = path.split('/').pop() ?? 'repo'
    const rawName = segment.replace(/\.git$/i, '').trim() || 'repo'
    const slug = rawName
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '')
      .slice(0, 128) || 'repo'

    const name = slug
      .split('-')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ')

    return { name, slug }
  } catch {
    return { name: '', slug: '' }
  }
}

const defaultForm: CreateCodeRepositoryRequest = {
  name: '',
  slug: '',
  git_repo_url: '',
  git_username: '',
  git_password: '',
}

export function CreateCodeRepositoryDialog({ open, onOpenChange, projectId, onSuccess }: CreateCodeRepositoryDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<CreateCodeRepositoryRequest>(defaultForm)

  const handleGitRepoUrlChange = (value: string) => {
    const parsed = parseRepoUrl(value)
    setForm((prev) => ({
      ...prev,
      git_repo_url: value,
      name: parsed.name || prev.name,
      slug: parsed.slug || prev.slug,
    }))
  }

  const createMutation = useMutation({
    mutationFn: () => codeRepositoriesApi.create(projectId, form),
    onSuccess: (repo) => {
      queryClient.invalidateQueries({ queryKey: ['code-repositories', projectId] })
      toast.success('Code repository added')
      onOpenChange(false)
      setForm(defaultForm)
      onSuccess?.(repo)
    },
    onError: (err: unknown) => {
      const msg = err && typeof err === 'object' && 'response' in err
        ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
        : null
      toast.error(msg || 'Failed to add code repository')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.git_repo_url?.trim()) {
      toast.error('Git Repository URL is required')
      return
    }
    createMutation.mutate()
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Code Repository</DialogTitle>
            <DialogDescription>
              Add a new code repository to your project.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>Git Repository URL *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="https://github.com/user/repo.git"
                  value={form.git_repo_url}
                  onChange={(e) => handleGitRepoUrlChange(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Name</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="From URL"
                    value={form.name ?? ''}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Slug</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="From URL"
                    value={form.slug ?? ''}
                    onChange={(e) => setForm({ ...form, slug: e.target.value })}
                  />
                </FieldContent>
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Git Username</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder=""
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
                    placeholder=""
                    value={form.git_password ?? ''}
                    onChange={(e) => setForm({ ...form, git_password: e.target.value })}
                  />
                </FieldContent>
              </Field>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Add
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
