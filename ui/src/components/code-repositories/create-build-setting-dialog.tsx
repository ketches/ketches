import * as React from "react"

import { codeRepositoriesApi, type CreateBuildSettingRequest } from "@/api/code-repositories"
import { BuildSettingSheet, type BuildSettingSheetSubmitPayload } from "@/components/code-repositories/build-setting-dialog"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
import { toast } from "sonner"

interface CreateBuildSettingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  onSuccess?: () => void
}

export function CreateBuildSettingDialog({ open, onOpenChange, repoId, onSuccess }: CreateBuildSettingDialogProps) {
  const queryClient = useQueryClient()

  const { data: repo } = useQuery({
    queryKey: ["code-repository", repoId],
    queryFn: () => codeRepositoriesApi.get(repoId),
    enabled: open && !!repoId,
  })

  const { data: registries = [] } = useQuery({
    queryKey: ["code-repository-registries", repoId],
    queryFn: () => codeRepositoriesApi.listContainerRegistries(repoId),
    enabled: open && !!repoId,
  })

  const createMutation = useMutation({
    mutationFn: (payload: CreateBuildSettingRequest) => codeRepositoriesApi.createBuildSetting(repoId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["build-settings", repoId] })
      toast.success("Build setting added")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error(error.response?.data?.error || "Failed to add build setting")
    },
  })

  const handleSubmit = React.useCallback((payload: BuildSettingSheetSubmitPayload) => {
    createMutation.mutate(payload)
  }, [createMutation])

  return (
    <BuildSettingSheet
      mode="create"
      open={open}
      onOpenChange={onOpenChange}
      repoId={repoId}
      repoSlug={repo?.slug}
      registries={registries}
      isPending={createMutation.isPending}
      onSubmit={handleSubmit}
    />
  )
}
