import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Key, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { CodeRepository } from "@/api/code-repositories"
import { codeRepositoriesApi, type CreateCodeRepositoryRequest } from "@/api/code-repositories"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import {
  deriveRepoDefaults,
  toRepositoryNameSlug,
} from "./code-repository-dialog.utils"

interface CreateCodeRepositoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: (repo: CodeRepository) => void
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
  const [showCredentials, setShowCredentials] = React.useState(false)
  const [hasEditedName, setHasEditedName] = React.useState(false)
  const [hasEditedSlug, setHasEditedSlug] = React.useState(false)

  const handleGitRepoUrlChange = (value: string) => {
    const parsed = deriveRepoDefaults(value)
    setForm((prev) => ({
      ...prev,
      git_repo_url: value,
      name: hasEditedName ? prev.name : parsed.name,
      slug: hasEditedSlug ? prev.slug : parsed.slug,
    }))
  }

  const createMutation = useMutation({
    mutationFn: () => codeRepositoriesApi.create(projectId, form),
    onSuccess: (repo) => {
      queryClient.invalidateQueries({ queryKey: ['code-repositories', projectId] })
      toast.success('Code repository added')
      onOpenChange(false)
      setForm(defaultForm)
      setShowCredentials(false)
      setHasEditedName(false)
      setHasEditedSlug(false)
      onSuccess?.(repo)
    },
    onError: (err: unknown) => {
      const msg = err && typeof err === 'object' && 'response' in err
        ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
        : null
      toast.error(msg || 'Failed to add code repository')
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
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
              <FieldLabel htmlFor="git-repo-url">Git Repository URL *</FieldLabel>
              <FieldContent>
                <InputGroup>
                  <InputGroupInput id="git-repo-url"
                    name="git_repo_url"
                    placeholder="https://github.com/user/repo.git"
                    value={form.git_repo_url}
                    onChange={(e) => handleGitRepoUrlChange(e.target.value)}
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
                      name="git_username"
                      placeholder="Git username"
                      value={form.git_username ?? ''}
                      onChange={(e) => setForm({ ...form, git_username: e.target.value })}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel htmlFor="git-password">Git Password / Token</FieldLabel>
                  <FieldContent>
                    <Input
                      id="git-password"
                      name="git_password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="Git password or token"
                      value={form.git_password ?? ''}
                      onChange={(e) => setForm({ ...form, git_password: e.target.value })}
                    />
                  </FieldContent>
                </Field>
              </div>
            )}
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="name">Name</FieldLabel>
                <FieldContent>
                  <Input
                    id="name"
                    name="name"
                    placeholder="From URL"
                    value={form.name ?? ''}
                    onChange={(e) => {
                      const name = e.target.value
                      setHasEditedName(true)
                      setForm((prev) => ({
                        ...prev,
                        name,
                        slug: hasEditedSlug ? prev.slug : toRepositoryNameSlug(name),
                      }))
                    }}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="slug">Slug</FieldLabel>
                <FieldContent>
                  <Input
                    id="slug"
                    name="slug"
                    placeholder="From URL"
                    value={form.slug ?? ''}
                    onChange={(e) => {
                      setHasEditedSlug(true)
                      setForm({ ...form, slug: e.target.value })
                    }}
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
