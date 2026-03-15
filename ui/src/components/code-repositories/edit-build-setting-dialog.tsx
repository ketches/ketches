import { codeRepositoriesApi, type BuildSetting, type UpdateBuildSettingRequest } from "@/api/code-repositories"
import { BuildSettingSheet, type BuildSettingSheetSubmitPayload } from "@/components/code-repositories/build-setting-sheet"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
import * as React from "react"
import { toast } from "sonner"

interface EditBuildSettingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  setting: BuildSetting | null
  onSuccess?: () => void
}

export function EditBuildSettingDialog({ open, onOpenChange, repoId, setting: setting, onSuccess }: EditBuildSettingDialogProps) {
  const queryClient = useQueryClient()

  const { data: registries = [] } = useQuery({
    queryKey: ['code-repository-registries', repoId],
    queryFn: () => codeRepositoriesApi.listContainerRegistries(repoId),
    enabled: open && !!repoId,
  })

  const updateMutation = useMutation({
    mutationFn: (payload: UpdateBuildSettingRequest) => codeRepositoriesApi.updateBuildSetting(repoId, setting!.id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-settings', repoId] })
      toast.success('Build setting updated')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to update build setting')
    },
  })

  const handleSubmit = React.useCallback((payload: BuildSettingSheetSubmitPayload) => {
    updateMutation.mutate(payload)
  }, [updateMutation])

  if (!setting) return null

  return (
    <BuildSettingSheet
      mode="edit"
      open={open}
      onOpenChange={onOpenChange}
      repoId={repoId}
      registries={registries}
      setting={setting}
      isPending={updateMutation.isPending}
      onSubmit={handleSubmit}
    />
  )
}
