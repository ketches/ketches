import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2, Plus, X } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi } from "@/api/apps"
import { codeRepositoriesApi, type CodeRepositoryBuildConfig } from "@/api/code-repositories"
import { envsApi, type Env } from "@/api/envs"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

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

  const [selectedConfigId, setSelectedConfigId] = React.useState<string>("")
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

  const { data: buildConfigs = [] } = useQuery({
    queryKey: ["code-repository-build-configs", repoId],
    queryFn: () => codeRepositoriesApi.listBuildConfigs(repoId),
    enabled: !!repoId && open,
  })

  const { data: builds = [] } = useQuery({
    queryKey: ["code-repository-builds", repoId],
    queryFn: () => codeRepositoriesApi.listBuilds(repoId),
    enabled: !!repoId && !!preSelectedBuildId && open,
  })

  const { data: envs = [] } = useQuery({
    queryKey: ["envs", projectId],
    queryFn: () => envsApi.list(projectId),
    enabled: !!projectId && open,
  })

  const buildEnvs = (envs as Env[]).filter((e) => e.is_build_env)
  const deployEnvs = envs as Env[]

  const { data: appsInDeployEnv = [], isLoading: isLoadingApps } = useQuery({
    queryKey: ["apps", deployEnvId],
    queryFn: () => appsApi.list(deployEnvId),
    enabled: !!deployEnvId && (autoDeploy || isDeployMode) && open,
  })

  const existingRepoApps = (appsInDeployEnv as { id: string; name: string; slug: string; code_repository_id?: string }[]).filter(
    (a) => a.code_repository_id === repoId
  )

  const selectedConfig = (buildConfigs as CodeRepositoryBuildConfig[]).find((c) => c.id === selectedConfigId)
  const selectedBuild = preSelectedBuildId ? (builds as any[]).find((b) => b.id === preSelectedBuildId) : null
  const selectedBuildEnv = (envs as Env[]).find((e) => e.id === buildEnvId)
  const selectedDeployEnv = (envs as Env[]).find((e) => e.id === deployEnvId)
  const selectedApp = (existingRepoApps as any[]).find((a) => a.id === deployAppId)

  const isDeployMode = !!preSelectedBuildId
  const isBuildConfigMode = !!preSelectedConfigId && !preSelectedBuildId
  const isCodeRepoMode = !preSelectedConfigId && !preSelectedBuildId

  const isCreatingApp = !isLoadingApps && (showCreateApp || (!!deployEnvId && existingRepoApps.length === 0)) && !preSelectedDeployAppId

  React.useEffect(() => {
    if (open) {
      if (preSelectedConfigId) {
        setSelectedConfigId(preSelectedConfigId)
      } else if ((buildConfigs as CodeRepositoryBuildConfig[]).length > 0 && !selectedConfigId) {
        setSelectedConfigId((buildConfigs as CodeRepositoryBuildConfig[])[0].id)
      }

      if (preSelectedBuildId) {
        setSelectedBuildId(preSelectedBuildId)
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
  }, [open, preSelectedConfigId, preSelectedBuildId, preSelectedDeployEnvId, preSelectedDeployAppId, buildConfigs])

  React.useEffect(() => {
    if (selectedConfig && !gitRef) {
      setGitRef(selectedConfig.git_ref || "")
    }
  }, [selectedConfig, gitRef])

  React.useEffect(() => {
    if (open && buildEnvs.length > 0 && !buildEnvId) {
      setBuildEnvId(buildEnvs[0]?.id || (envs as Env[])[0]?.id || "")
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
        build_config_id: selectedConfigId,
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
      queryClient.invalidateQueries({ queryKey: ["code-repository-builds", repoId] })
      onOpenChange(false)
      resetForm()
      toast.success(autoDeploy ? "Build triggered with auto-deploy" : "Build triggered")
    },
    onError: (err: any) => {
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
      queryClient.invalidateQueries({ queryKey: ["code-repository-builds", repoId] })
      queryClient.invalidateQueries({ queryKey: ["code-repository-deployments", repoId] })
      onOpenChange(false)
      resetForm()
      toast.success("Deployed successfully")
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.error || "Failed to deploy")
    },
  })

  const resetForm = () => {
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
  }

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
      if (!selectedConfigId) {
        toast.error("Please select a build config")
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
    if (isBuildConfigMode) return "Trigger Build"
    return "Build & Deploy"
  }

  const getDialogDescription = () => {
    if (isDeployMode) return "Deploy this build to an environment"
    if (isBuildConfigMode) return "Trigger a new build with optional auto-deploy"
    return "Configure and trigger a new build for this repository"
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{getDialogTitle()}</DialogTitle>
          <DialogDescription>{getDialogDescription()}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {!isDeployMode && (
            <>
              {isCodeRepoMode && (
                <Field>
                  <FieldLabel>Build Config *</FieldLabel>
                  <FieldContent>
                    <Select value={selectedConfigId} onValueChange={setSelectedConfigId}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select build config">
                          {selectedConfig?.name}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {(buildConfigs || []).map((config) => (
                          <SelectItem key={config.id} value={config.id}>
                            {config.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FieldContent>
                </Field>
              )}

              {selectedConfigId && (
                <>
                  <Field>
                    <FieldLabel>Git Branch / Tag</FieldLabel>
                    <FieldContent>
                      <GitRefSelect
                        repoId={repoId}
                        value={gitRef}
                        onChange={setGitRef}
                        placeholder={selectedConfig?.git_ref || "Select branch or tag"}
                      />
                    </FieldContent>
                  </Field>

                  <Field>
                    <FieldLabel>Build Environment *</FieldLabel>
                    <FieldContent>
                      <Select value={buildEnvId} onValueChange={setBuildEnvId}>
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Select build environment">
                            {selectedBuildEnv ? `${selectedBuildEnv.name}${selectedBuildEnv.is_build_env ? ' (Build Env)' : ''}` : undefined}
                          </SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {(envs || []).map((env) => (
                            <SelectItem key={env.id} value={env.id}>
                              {env.name} {env.is_build_env && "(Build Env)"}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
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
                    <Select value={deployEnvId} onValueChange={setDeployEnvId}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select deploy environment">
                          {selectedDeployEnv?.name}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {(deployEnvs || []).map((env) => (
                          <SelectItem key={env.id} value={env.id}>
                            {env.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
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
                              <Select value={deployAppId} onValueChange={setDeployAppId}>
                                <SelectTrigger className="flex-1">
                                  <SelectValue placeholder="Select application">
                                    {selectedApp?.name}
                                  </SelectValue>
                                </SelectTrigger>
                                <SelectContent>
                                  {(existingRepoApps || []).map((app) => (
                                    <SelectItem key={app.id} value={app.id}>
                                      {app.name}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
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
                        <div className="grid gap-4 border border-green-500/50 border-dashed rounded-md p-4">
                          <div className="flex items-center justify-between">
                            <FieldLabel>Create New Application</FieldLabel>
                            {existingRepoApps.length > 0 && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                  setShowCreateApp(false)
                                  setNewAppName("")
                                  setNewAppSlug("")
                                }}
                              >
                                <X className="h-3 w-3 mr-1" />
                                Cancel
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
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            {isDeployMode ? "Deploy" : autoDeploy ? "Build & Deploy" : "Build"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
