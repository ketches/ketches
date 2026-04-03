import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type PaginationState } from "@tanstack/react-table"
import { isAxiosError, type AxiosError } from "axios"
import { CheckCircle, FolderGit2, GalleryVerticalEnd, Hammer, Play, Rocket } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { codeRepositoriesApi, type BuildSetting } from "@/api/code-repositories"
import { operationLogsApi } from "@/api/operation-logs"
import { projectsApi } from "@/api/projects"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

export interface RetryBuildPayload {
  id: string
  build_setting_id: string
  build_env_id: string
  git_ref?: string
}

export function useCodeRepositoryDetail(repoId: string | undefined) {
  const queryClient = useQueryClient()
  const { activeProjectId, activeProjectName, setActiveContextWithNames } = useProjectStore()
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")
  const hasSyncedProjectFromRepoRef = React.useRef(false)

  const [operationLogsPagination, setOperationLogsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [retryingBuildId, setRetryingBuildId] = React.useState<string | null>(null)

  const { data: repo, isLoading, error: repoError } = useQuery({
    queryKey: ["code-repository", repoId],
    queryFn: () => codeRepositoriesApi.get(repoId!),
    enabled: !!repoId,
  })

  React.useEffect(() => {
    hasSyncedProjectFromRepoRef.current = false
  }, [repoId])

  React.useEffect(() => {
    if (!hasSyncedProjectFromRepoRef.current && repo?.project_id) {
      if (activeProjectId !== repo.project_id) {
        projectsApi.get(repo.project_id).then((project) => {
          setActiveContextWithNames(repo.project_id, project.name, null, null)
        }).catch(() => {
          setActiveContextWithNames(repo.project_id, null, null, null)
        })
      }
      hasSyncedProjectFromRepoRef.current = true
    }
  }, [repo?.project_id, activeProjectId, setActiveContextWithNames])

  const { data: repos = [] } = useQuery({
    queryKey: ["code-repositories-simple", repo?.project_id],
    queryFn: () => codeRepositoriesApi.listSimple(repo!.project_id),
    enabled: !!repo?.project_id,
  })

  const { data: buildSettings = [], isLoading: buildSettingsLoading } = useQuery({
    queryKey: ["build-settings", repoId],
    queryFn: () => codeRepositoriesApi.listBuildSettings(repoId!),
    enabled: !!repoId,
  })

  const { data: builds = [], isLoading: buildsLoading } = useQuery({
    queryKey: ["builds", repoId],
    queryFn: () => codeRepositoriesApi.listBuilds(repoId!),
    enabled: !!repoId,
    refetchInterval: 5000,
  })

  const { data: deployments = [], isLoading: deploymentsLoading } = useQuery({
    queryKey: ["code-repository-deployments", repoId],
    queryFn: () => codeRepositoriesApi.listDeployments(repoId!),
    enabled: !!repoId,
    refetchInterval: 5000,
  })

  const { data: operationLogsResponse, isLoading: operationLogsLoading, isFetching: operationLogsFetching } = useQuery({
    queryKey: ["repo-operation-logs", repoId, operationLogsPagination.pageIndex, operationLogsPagination.pageSize],
    queryFn: () => operationLogsApi.listCodeRepositoryOperationLogs(repoId!, {
      page: operationLogsPagination.pageIndex + 1,
      page_size: operationLogsPagination.pageSize,
    }),
    enabled: !!repoId,
  })

  const retryBuildMutation = useMutation({
    mutationFn: (payload: RetryBuildPayload) => codeRepositoriesApi.triggerBuild(repoId!, {
      build_setting_id: payload.build_setting_id,
      build_env_id: payload.build_env_id,
      git_ref: payload.git_ref,
    }),
    onMutate: (payload) => {
      setRetryingBuildId(payload.id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["builds", repoId] })
      toast.success("Build retry triggered")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error(error.response?.data?.error || "Failed to retry build")
    },
    onSettled: () => {
      setRetryingBuildId(null)
    },
  })

  const deleteBuildSettingMutation = useMutation({
    mutationFn: (settingId: string) => codeRepositoriesApi.deleteBuildSetting(repoId!, settingId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["build-settings", repoId] })
      toast.success("Build setting removed")
    },
    onError: (error: unknown) => {
      const message =
        error && typeof error === "object" && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response?.data?.error
          : null
      toast.error(message || "Failed to remove build setting")
    },
  })

  const safeRepos = Array.isArray(repos) ? repos : []
  const settingNameById = React.useCallback((id: string) => {
    return buildSettings.find((setting) => setting.id === id)?.name ?? id
  }, [buildSettings])

  const totalBuilds = builds.length
  const successfulBuilds = builds.filter((build) => build.status === "succeeded").length
  const successRate = totalBuilds > 0 ? (successfulBuilds / totalBuilds) * 100 : 0
  const totalDeployments = deployments.length
  const totalBuildSettings = buildSettings.length

  const breadcrumbs: BreadcrumbItem[] = isAdmin
    ? [
      { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
      {
        label: activeProjectName ?? "Project",
        icon: GalleryVerticalEnd,
        href: `/projects/${activeProjectId}`,
      },
      {
        label: repo?.name ?? "Repository",
        icon: FolderGit2,
      },
    ]
    : [
      {
        label: "Code Repositories",
        href: "/code-repositories",
        icon: FolderGit2,
      },
      {
        label: repo?.name ?? "Repository",
        icon: FolderGit2,
      },
    ]

  const overviewStats = [
    {
      title: "Build Settings",
      value: totalBuildSettings,
      icon: Hammer,
      description: "Configurations defined",
      color: "purple" as const,
    },
    {
      title: "Total Builds",
      value: totalBuilds,
      icon: Play,
      description: "All time builds",
      color: "blue" as const,
    },
    {
      title: "Success Rate",
      value: totalBuilds > 0 ? `${successRate.toFixed(0)}%` : "N/A",
      icon: CheckCircle,
      description: "Build success rate",
      color: totalBuilds > 0 ? (successRate >= 90 ? "green" : successRate >= 70 ? "orange" : "red") : "gray" as const,
    },
    {
      title: "Deployments",
      value: totalDeployments,
      icon: Rocket,
      description: "Total deployments",
      color: "sky" as const,
    },
  ]

  return {
    repo,
    isLoading,
    repoError,
    isRepoForbidden: isAxiosError(repoError) && repoError.response?.status === 403,
    safeRepos,
    breadcrumbs,
    buildSettings,
    buildSettingsLoading,
    builds,
    buildsLoading,
    deployments,
    deploymentsLoading,
    operationLogsResponse,
    operationLogsLoading,
    operationLogsFetching,
    operationLogsPagination,
    setOperationLogsPagination,
    retryBuildMutation,
    retryingBuildId,
    deleteBuildSettingMutation,
    settingNameById,
    overviewStats,
  }
}
