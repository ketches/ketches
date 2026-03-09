import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2, Plus, X } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { SimpleApp } from "@/api/apps"
import { codeRepositoriesApi, type BuildSetting } from "@/api/code-repositories"
import { envsApi, type Env } from "@/api/envs"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import type { AxiosError } from "axios"
import { ColorBadge } from "../shared/color-badge"
import { Item, ItemContent, ItemDescription, ItemTitle } from "../ui/item"

interface UnifiedBuildDeployDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void

  repoId: string
  projectId: string

  preSelectedConfigId?: string
  preSelectedBuildId?: string

  preSelectedDeployEnvId?: string
  preSelectedDeployAppId?: string
}

export function UnifiedBuildDeployDialog({
  open,
  onOpenChange,
  repoId,
  projectId,
  preSelectedConfigId,
  preSelectedBuildId,
  preSelectedDeployEnvId,
  preSelectedDeployAppId,
}: UnifiedBuildDeployDialogProps) {
  const queryClient = useQueryClient()

  const [selectedBuildSettingId, setSelectedConfigId] = React.useState<string>("")
  const [gitRef, setGitRef] = React.useState<string>("")
  const [buildEnvId, setBuildEnvId] = React.useState<string>("")
  const [autoDeploy, setAutoDeploy] = React.useState<boolean>(false)
  const [deployEnvId, setDeployEnvId] = React.useState<string>("")
  const [deployAppId, setDeployAppId] = React.useState<string>("")
  const [showCreateApp, setShowCreateApp] = React.useState<boolean>(false)
  const [newAppName, setNewAppName] = React.useState<string>("")
  const [newAppSlug, setNewAppSlug] = React.useState<string>("")

  const { data: repo } = useQuery({
    queryKey: ["code-repository", repoId],
    queryFn: () => codeRepositoriesApi.get(repoId),
    enabled: !!repoId && open,
  })

  const { data: buildSettings = [] } = useQuery({
    queryKey: ["build-settings", repoId],
    queryFn: () => codeRepositoriesApi.listBuildSettings(repoId),
    enabled: !!repoId && open,
  })

  const { data: _builds = [] } = useQuery({
    queryKey: ["builds", repoId],
    queryFn: () => codeRepositoriesApi.listBuilds(repoId),
    enabled: !!repoId && !!preSelectedBuildId && open,
  })

  const { data: envs = [] } = useQuery({
    queryKey: ["envs-simple", projectId],
    queryFn: () => envsApi.listSimpleByProject(projectId!),
    enabled: !!projectId && open,
  })

  const buildEnvs = envs.filter((e: Env) => e.is_build_env)
  const deployEnvs = envs

  const isDeployMode = !!preSelectedBuildId

  const effectiveBuildSettingId = React.useMemo(() => {
    if (!isDeployMode) return selectedBuildSettingId
    const currentBuild = _builds.find((build) => build.id === preSelectedBuildId)
    return currentBuild?.build_setting_id || ""
  }, [_builds, isDeployMode, preSelectedBuildId, selectedBuildSettingId])

  const { data: deployedAppsInEnv = [], isLoading: isLoadingApps } = useQuery({
    queryKey: ["deployed-apps-by-env", repoId, deployEnvId, effectiveBuildSettingId],
    queryFn: () => codeRepositoriesApi.listDeployedAppsByEnv(repoId, deployEnvId!, effectiveBuildSettingId),
    enabled: !!deployEnvId && !!effectiveBuildSettingId && (autoDeploy || isDeployMode) && open,
  })

  const existingRepoApps: SimpleApp[] = React.useMemo(
    () => deployedAppsInEnv.map((app) => ({
      id: app.id,
      name: app.name,
      slug: "",
      description: "",
      status: "",
      code_repository_id: repoId,
    })),
    [deployedAppsInEnv, repoId]
  )

  const selectedConfig = (buildSettings as BuildSetting[]).find((c) => c.id === selectedBuildSettingId)
  const isBuildSettingMode = !!preSelectedConfigId && !preSelectedBuildId
  const isCodeRepoMode = !preSelectedConfigId && !preSelectedBuildId

  const isCreatingApp = !isLoadingApps && (showCreateApp || (!!deployEnvId && existingRepoApps.length === 0)) && !preSelectedDeployAppId

  const resetForm = React.useCallback(() => {
    if (!preSelectedConfigId) setSelectedConfigId("")
    setGitRef("")
    setBuildEnvId("")
    if (!isDeployMode && !preSelectedDeployEnvId) {
      setAutoDeploy(false)
      setDeployEnvId("")
    }
    if (!preSelectedDeployAppId) setDeployAppId("")
    setShowCreateApp(false)
    setNewAppName("")
    setNewAppSlug("")
  }, [preSelectedConfigId, isDeployMode, preSelectedDeployEnvId, preSelectedDeployAppId])

  React.useEffect(() => {
    if (open) {
      if (preSelectedConfigId) {
        setSelectedConfigId(preSelectedConfigId)
      } else if ((buildSettings as BuildSetting[]).length > 0 && !selectedBuildSettingId) {
        setSelectedConfigId((buildSettings as BuildSetting[])[0].id)
      }

      if (preSelectedBuildId) {
        setAutoDeploy(true)
      }

      if (preSelectedDeployEnvId) {
        setDeployEnvId(preSelectedDeployEnvId)
        setAutoDeploy(true)
      }

      if (preSelectedDeployAppId) {
        setDeployAppId(preSelectedDeployAppId)
      }
    } else {
      resetForm()
    }
  }, [open, preSelectedConfigId, preSelectedBuildId, preSelectedDeployEnvId, preSelectedDeployAppId, buildSettings, resetForm, selectedBuildSettingId])

  React.useEffect(() => {
    if (selectedConfig && !gitRef) {
      setGitRef(selectedConfig.git_ref || "")
    }
  }, [selectedConfig, gitRef])

  React.useEffect(() => {
    if (open && buildEnvs.length > 0 && !buildEnvId) {
      setBuildEnvId(buildEnvs[0]?.id || envs[0]?.id || "")
    }
  }, [open, buildEnvs, buildEnvId, envs])

  React.useEffect(() => {
    if (open && !autoDeploy && !isDeployMode && !preSelectedDeployEnvId) {
      setDeployEnvId("")
      setDeployAppId("")
      setShowCreateApp(false)
    }
  }, [open, autoDeploy, isDeployMode, preSelectedDeployEnvId])

  React.useEffect(() => {
    if (open && !preSelectedDeployAppId && deployEnvId && deployEnvId !== preSelectedDeployEnvId) {
      setDeployAppId("")
      setShowCreateApp(false)
    }
  }, [open, deployEnvId, preSelectedDeployAppId, preSelectedDeployEnvId])

  React.useEffect(() => {
    if (existingRepoApps.length === 1 && !deployAppId && !showCreateApp && !preSelectedDeployAppId) {
      setDeployAppId(existingRepoApps[0].id)
    }
  }, [existingRepoApps, deployAppId, showCreateApp, preSelectedDeployAppId])

  React.useEffect(() => {
    if (isCreatingApp && repo && !newAppName && !newAppSlug) {
      setNewAppName(repo.name)
      setNewAppSlug(repo.slug)
    }
  }, [isCreatingApp, repo, newAppName, newAppSlug])

  const triggerBuildMutation = useMutation({
    mutationFn: () => {
      const finalDeployEnvId = deployEnvId || preSelectedDeployEnvId
      const finalDeployAppId = deployAppId || preSelectedDeployAppId

      const payload: any = {
        build_setting_id: selectedBuildSettingId,
        build_env_id: buildEnvId,
      }

      if (gitRef) payload.git_ref = gitRef

      if (autoDeploy) {
        payload.auto_deploy = true
        payload.deploy_env_id = finalDeployEnvId

        if (isCreatingApp) {
          payload.deploy_app_name = newAppName
          payload.deploy_app_slug = newAppSlug
        } else {
          payload.deploy_app_id = finalDeployAppId
        }
      }

      return codeRepositoriesApi.triggerBuild(repoId, payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["builds", repoId] })
      onOpenChange(false)
      resetForm()
      toast.success(autoDeploy ? "Build triggered with auto-deploy" : "Build triggered")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || "Failed to trigger build")
    },
  })

  const deployBuildMutation = useMutation({
    mutationFn: () => {
      const finalDeployEnvId = deployEnvId || preSelectedDeployEnvId
      const finalDeployAppId = deployAppId || preSelectedDeployAppId

      const payload: any = {
        target_env_id: finalDeployEnvId,
      }

      if (isCreatingApp) {
        payload.name = newAppName
        payload.slug = newAppSlug
      } else {
        payload.app_id = finalDeployAppId
      }

      return codeRepositoriesApi.deployBuild(repoId, preSelectedBuildId!, payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["builds", repoId] })
      queryClient.invalidateQueries({ queryKey: ["code-repository-deployments", repoId] })
      onOpenChange(false)
      resetForm()
      toast.success("Deployed successfully")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || "Failed to deploy")
    },
  })

  const handleSubmit = () => {
    const finalDeployEnvId = deployEnvId || preSelectedDeployEnvId
    const finalDeployAppId = deployAppId || preSelectedDeployAppId

    if (isDeployMode) {
      if (!finalDeployEnvId) {
        toast.error("Please select a deploy environment")
        return
      }
      if (isCreatingApp) {
        if (!newAppName || !newAppSlug) {
          toast.error("Please provide app name and slug")
          return
        }
      } else if (!finalDeployAppId) {
        toast.error("Please select an app to deploy")
        return
      }
      deployBuildMutation.mutate()
    } else {
      if (!selectedBuildSettingId) {
        toast.error("Please select a build setting")
        return
      }
      if (!buildEnvId) {
        toast.error("Please select a build environment")
        return
      }
      if (autoDeploy) {
        if (!finalDeployEnvId) {
          toast.error("Please select a deploy environment")
          return
        }
        if (isCreatingApp) {
          if (!newAppName || !newAppSlug) {
            toast.error("Please provide app name and slug")
            return
          }
        } else if (!finalDeployAppId) {
          toast.error("Please select an app to deploy")
          return
        }
      }

      triggerBuildMutation.mutate()
    }
  }

  const getDialogTitle = () => {
    if (isDeployMode) return "Deploy Build"
    if (isBuildSettingMode) return "Trigger Build"
    return "Build & Deploy"
  }

  const getDialogDescription = () => {
    if (isDeployMode) return "Deploy this build to an environment"
    if (isBuildSettingMode) return "Trigger a new build with optional auto-deploy"
    return "Configure and trigger a new build for this repository"
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-w-2xl max-h-[90vh] overflow-y-auto gap-0">
        <DialogHeader>
          <DialogTitle>{getDialogTitle()}</DialogTitle>
          <DialogDescription>{getDialogDescription()}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {!isDeployMode && (
            <>
              {isCodeRepoMode && (
                <Field>
                  <FieldLabel>Build Setting *</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={selectedBuildSettingId}
                      onValueChange={(v) => v !== null && setSelectedConfigId(v)}
                      itemToStringLabel={(id) => buildSettings?.find((c: BuildSetting) => c.id === id)?.name ?? id ?? ""}
                    >
                      <ComboboxInput placeholder="Select build setting" />
                      <ComboboxContent>
                        <ComboboxList>
                          {(buildSettings || []).map((setting) => (
                            <ComboboxItem key={setting.id} value={setting.id}>
                              {setting.name}
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
              )}

              {selectedBuildSettingId && (
                <>
                  <Field>
                    <FieldLabel>Git Branch / Tag</FieldLabel>
                    <FieldContent>
                      <GitRefSelect
                        repoId={repoId}
                        value={gitRef}
                        onValueChange={(v) => v !== null && setGitRef(v)}
                        placeholder={selectedConfig?.git_ref || "Select branch or tag"}
                      />
                    </FieldContent>
                  </Field>

                  <Field>
                    <FieldLabel>Build Environment *</FieldLabel>
                    <FieldContent>
                      <Combobox
                        value={buildEnvId}
                        onValueChange={(v) => v !== null && setBuildEnvId(v)}
                        itemToStringLabel={(id) => envs?.find((e: Env) => e.id === id)?.name ?? id ?? ""}
                      >
                        <ComboboxInput placeholder="Select build environment" />
                        <ComboboxContent>
                          <ComboboxList>
                            {(envs || []).map((env: Env) => (
                              <ComboboxItem key={env.id} value={env.id}>
                                <Item size="xs" className="p-0">
                                  <ItemContent>
                                    <ItemTitle><>{env.name}{env.is_build_env && <ColorBadge>Build</ColorBadge>}</></ItemTitle>
                                    <ItemDescription>{env.slug}</ItemDescription>
                                  </ItemContent>
                                </Item>
                              </ComboboxItem>
                            ))}
                          </ComboboxList>
                        </ComboboxContent>
                      </Combobox>
                    </FieldContent>
                  </Field>

                  {!preSelectedDeployEnvId && (
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="auto-deploy"
                        checked={autoDeploy}
                        onCheckedChange={(checked) => setAutoDeploy(checked === true)}
                      />
                      <label htmlFor="auto-deploy" className="cursor-pointer">
                        Auto deploy after successful build
                      </label>
                    </div>
                  )}
                </>
              )}
            </>
          )}

          {(autoDeploy || isDeployMode) && (
            <>
              {!preSelectedDeployEnvId && (
                <Field>
                  <FieldLabel>Deploy Environment *</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={deployEnvId}
                      onValueChange={(v) => v !== null && setDeployEnvId(v)}
                      itemToStringLabel={(id) => deployEnvs?.find((e: Env) => e.id === id)?.name ?? id ?? ""}
                    >
                      <ComboboxInput placeholder="Select deploy environment" />
                      <ComboboxContent>
                        <ComboboxList>
                          {(deployEnvs || []).map((env: Env) => (
                            <ComboboxItem key={env.id} value={env.id}>
                              <Item size="xs" className="p-0">
                                <ItemContent>
                                  <ItemTitle>{env.name}</ItemTitle>
                                  <ItemDescription>{env.slug}</ItemDescription>
                                </ItemContent>
                              </Item>
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
              )}

              {deployEnvId && !preSelectedDeployAppId && (
                <>
                  {isLoadingApps ? (
                    <div className="flex items-center justify-center py-4">
                      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                  ) : (
                    <>
                      {!isCreatingApp && existingRepoApps.length > 0 && (
                        <Field>
                          <FieldLabel>Deploy to Application *</FieldLabel>
                          <FieldContent>
                            <div className="flex gap-2">
                              <Combobox
                                value={deployAppId}
                                onValueChange={(v) => v !== null && setDeployAppId(v)}
                                itemToStringLabel={(id) => existingRepoApps?.find((a: SimpleApp) => a.id === id)?.name ?? id ?? ""}
                              >
                                <ComboboxInput placeholder="Select application" className="flex-1" />
                                <ComboboxContent>
                                  <ComboboxList>
                                    {(existingRepoApps || []).map((app) => (
                                      <ComboboxItem key={app.id} value={app.id}>
                                        {app.name}
                                      </ComboboxItem>
                                    ))}
                                  </ComboboxList>
                                </ComboboxContent>
                              </Combobox>
                              <Button
                                variant="outline"
                                size="icon"
                                onClick={() => {
                                  setShowCreateApp(true)
                                  setDeployAppId("")
                                }}
                                title="Create new application"
                              >
                                <Plus />
                              </Button>
                            </div>
                          </FieldContent>
                        </Field>
                      )}

                      {isCreatingApp && (
                        <div className="grid gap-4 border border-primary/20 border-dashed rounded-md p-4">
                          <div className="flex items-center justify-between">
                            <FieldLabel>Create New Application</FieldLabel>
                            {existingRepoApps.length > 0 && (
                              <Button
                                variant="destructive"
                                size="icon-xs"
                                onClick={() => {
                                  setShowCreateApp(false)
                                  setNewAppName("")
                                  setNewAppSlug("")
                                }}
                              >
                                <X />
                              </Button>
                            )}
                          </div>

                          <Field>
                            <FieldLabel>Application Name *</FieldLabel>
                            <FieldContent>
                              <Input
                                value={newAppName}
                                onChange={(e) => setNewAppName(e.target.value)}
                                placeholder="My Application"
                              />
                            </FieldContent>
                          </Field>

                          <Field>
                            <FieldLabel>Application Slug *</FieldLabel>
                            <FieldContent>
                              <Input
                                value={newAppSlug}
                                onChange={(e) => setNewAppSlug(e.target.value)}
                                placeholder="my-application"
                              />
                            </FieldContent>
                          </Field>
                        </div>
                      )}
                    </>
                  )}
                </>
              )}
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={triggerBuildMutation.isPending || deployBuildMutation.isPending}
          >
            {(triggerBuildMutation.isPending || deployBuildMutation.isPending) && (
              <Loader2 className="h-4 w-4 animate-spin" />
            )}
            {isDeployMode ? "Deploy" : autoDeploy ? "Build & Deploy" : "Build"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
