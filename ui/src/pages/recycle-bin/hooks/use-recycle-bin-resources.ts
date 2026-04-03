import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type PaginationState } from "@tanstack/react-table"
import { AxiosError } from "axios"
import * as React from "react"
import { toast } from "sonner"

import { recycleBinApi, type RecycleBinApp, type RecycleBinCodeRepo, type RecycleBinEnv, type RecycleBinProject, type RecycleBinUser } from "@/api/recycle-bin"
import { useDebounce } from "@/hooks/use-debounce"

export type RecycleBinTabKey = "projects" | "apps" | "envs" | "code-repos" | "users"
type RecycleBinResourceType = "project" | "app" | "env" | "code-repo" | "user"
type RowSelectionState = Record<string, boolean>

interface RecycleBinPaginationInfo {
  total?: number
}

export interface RecycleBinResourceState<T> {
  data: T[]
  isLoading: boolean
  isFetching: boolean
  refetch: () => Promise<unknown>
  pagination: PaginationState
  setPagination: React.Dispatch<React.SetStateAction<PaginationState>>
  rowSelection: RowSelectionState
  setRowSelection: React.Dispatch<React.SetStateAction<RowSelectionState>>
  selectedIds: string[]
  paginationInfo?: RecycleBinPaginationInfo
}

interface UseRecycleBinResourcesOptions {
  activeTab: RecycleBinTabKey
  isAdmin: boolean
}

function getSelectedIds<T extends { id: string }>(rowSelection: RowSelectionState, items: T[]): string[] {
  return Object.keys(rowSelection)
    .filter((key) => rowSelection[key])
    .map((index) => items[Number.parseInt(index, 10)]?.id)
    .filter((id): id is string => Boolean(id))
}

function getErrorDescription(error: unknown): string {
  if (typeof error !== "object" || error === null) {
    return "An unknown error occurred"
  }

  const response = (error as { response?: { data?: { error?: unknown } } }).response
  if (typeof response?.data?.error === "string" && response.data.error.length > 0) {
    return response.data.error
  }

  if ("message" in error && typeof error.message === "string" && error.message.length > 0) {
    return error.message
  }

  return "An unknown error occurred"
}

export function useRecycleBinResources({ activeTab, isAdmin }: UseRecycleBinResourcesOptions) {
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  const [selectedAppRows, setSelectedAppRows] = React.useState<RowSelectionState>({})
  const [selectedEnvRows, setSelectedEnvRows] = React.useState<RowSelectionState>({})
  const [selectedProjectRows, setSelectedProjectRows] = React.useState<RowSelectionState>({})
  const [selectedCodeRepoRows, setSelectedCodeRepoRows] = React.useState<RowSelectionState>({})
  const [selectedUserRows, setSelectedUserRows] = React.useState<RowSelectionState>({})

  const [conflictDialogOpen, setConflictDialogOpen] = React.useState(false)
  const [conflictApps, setConflictApps] = React.useState<RecycleBinApp[]>([])
  const [restoringItemId, setRestoringItemId] = React.useState<string | null>(null)
  const [deletingItemId, setDeletingItemId] = React.useState<string | null>(null)

  const [appsPagination, setAppsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [envsPagination, setEnvsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [projectsPagination, setProjectsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [usersPagination, setUsersPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [codeReposPagination, setCodeReposPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: appsResponse, isLoading: appsLoading, isFetching: appsFetching, refetch: refetchApps } = useQuery({
    queryKey: ["recycle-bin-apps", debouncedSearch, appsPagination.pageIndex, appsPagination.pageSize],
    queryFn: () => recycleBinApi.listApps(undefined, {
      search: debouncedSearch,
      page: appsPagination.pageIndex + 1,
      page_size: appsPagination.pageSize,
    }),
  })
  const apps = React.useMemo(() => appsResponse?.items ?? [], [appsResponse])

  const { data: envsResponse, isLoading: envsLoading, isFetching: envsFetching, refetch: refetchEnvs } = useQuery({
    queryKey: ["recycle-bin-envs", debouncedSearch, envsPagination.pageIndex, envsPagination.pageSize],
    queryFn: () => recycleBinApi.listEnvs(undefined, {
      search: debouncedSearch,
      page: envsPagination.pageIndex + 1,
      page_size: envsPagination.pageSize,
    }),
  })
  const envs = React.useMemo(() => envsResponse?.items ?? [], [envsResponse])

  const { data: projectsResponse, isLoading: projectsLoading, isFetching: projectsFetching, refetch: refetchProjects } = useQuery({
    queryKey: ["recycle-bin-projects", debouncedSearch, projectsPagination.pageIndex, projectsPagination.pageSize],
    queryFn: () => recycleBinApi.listProjects({
      search: debouncedSearch,
      page: projectsPagination.pageIndex + 1,
      page_size: projectsPagination.pageSize,
    }),
  })
  const projects = React.useMemo(() => projectsResponse?.items ?? [], [projectsResponse])

  const { data: usersResponse, isLoading: usersLoading, isFetching: usersFetching, refetch: refetchUsers } = useQuery({
    queryKey: ["recycle-bin-users", debouncedSearch, usersPagination.pageIndex, usersPagination.pageSize],
    queryFn: () => recycleBinApi.listUsers({
      search: debouncedSearch,
      page: usersPagination.pageIndex + 1,
      page_size: usersPagination.pageSize,
    }),
    enabled: isAdmin,
  })
  const users = React.useMemo(() => usersResponse?.items ?? [], [usersResponse])

  const { data: codeReposResponse, isLoading: codeReposLoading, isFetching: codeReposFetching, refetch: refetchCodeRepos } = useQuery({
    queryKey: ["recycle-bin-code-repos", debouncedSearch, codeReposPagination.pageIndex, codeReposPagination.pageSize],
    queryFn: () => recycleBinApi.listCodeRepos(undefined, {
      search: debouncedSearch,
      page: codeReposPagination.pageIndex + 1,
      page_size: codeReposPagination.pageSize,
    }),
  })
  const codeRepos = React.useMemo(() => codeReposResponse?.items ?? [], [codeReposResponse])

  const selectedAppIds = React.useMemo(() => getSelectedIds(selectedAppRows, apps), [selectedAppRows, apps])
  const selectedEnvIds = React.useMemo(() => getSelectedIds(selectedEnvRows, envs), [selectedEnvRows, envs])
  const selectedProjectIds = React.useMemo(() => getSelectedIds(selectedProjectRows, projects), [selectedProjectRows, projects])
  const selectedCodeRepoIds = React.useMemo(() => getSelectedIds(selectedCodeRepoRows, codeRepos), [selectedCodeRepoRows, codeRepos])
  const selectedUserIds = React.useMemo(() => getSelectedIds(selectedUserRows, users), [selectedUserRows, users])

  const clearResourceSelection = React.useCallback((type: RecycleBinResourceType) => {
    if (type === "project") {
      setSelectedProjectRows({})
      return
    }
    if (type === "app") {
      setSelectedAppRows({})
      return
    }
    if (type === "env") {
      setSelectedEnvRows({})
      return
    }
    if (type === "code-repo") {
      setSelectedCodeRepoRows({})
      return
    }
    setSelectedUserRows({})
  }, [])

  const restoreAppsMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.restoreApps(ids),
    onSuccess: () => {
      toast.success("Applications restored")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-apps"] })
      clearResourceSelection("app")
      setRestoringItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to restore applications", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteAppsMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.permanentlyDeleteApps(ids),
    onSuccess: () => {
      toast.success("Applications permanently deleted")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-apps"] })
      clearResourceSelection("app")
      setDeletingItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to delete applications", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeletingItemId(null)
    },
  })

  const restoreEnvsMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.restoreEnvs(ids),
    onSuccess: () => {
      toast.success("Environments restored")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-envs"] })
      clearResourceSelection("env")
      setRestoringItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to restore environments", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteEnvsMutation = useMutation<unknown, Error, string[]>({
    mutationFn: async (ids) => {
      for (const id of ids) {
        const conflicts = await recycleBinApi.checkEnvDeletionConflicts(id)
        if (conflicts.apps.length > 0) {
          setConflictApps(conflicts.apps)
          setConflictDialogOpen(true)
          throw new Error("Environment has deleted applications")
        }
      }
      return recycleBinApi.permanentlyDeleteEnvs(ids)
    },
    onSuccess: () => {
      toast.success("Environments permanently deleted")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-envs"] })
      clearResourceSelection("env")
      setDeletingItemId(null)
    },
    onError: (error) => {
      if (error.message !== "Environment has deleted applications") {
        toast.error("Failed to delete environments", {
          description: getErrorDescription(error),
        })
      }
      setDeletingItemId(null)
    },
  })

  const restoreProjectsMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.restoreProjects(ids),
    onSuccess: () => {
      toast.success("Projects restored")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-projects"] })
      clearResourceSelection("project")
      setRestoringItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to restore projects", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteProjectsMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.permanentlyDeleteProjects(ids),
    onSuccess: () => {
      toast.success("Projects permanently deleted")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-projects"] })
      clearResourceSelection("project")
      setDeletingItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to delete projects", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeletingItemId(null)
    },
  })

  const restoreCodeReposMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.restoreCodeRepos(ids),
    onSuccess: () => {
      toast.success("Code repositories restored")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-code-repos"] })
      clearResourceSelection("code-repo")
      setRestoringItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to restore code repositories", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteCodeReposMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.permanentlyDeleteCodeRepos(ids),
    onSuccess: () => {
      toast.success("Code repositories permanently deleted")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-code-repos"] })
      clearResourceSelection("code-repo")
      setDeletingItemId(null)
    },
    onError: (error) => {
      toast.error("Failed to delete code repositories", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeletingItemId(null)
    },
  })

  const restoreUsersMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.restoreUsers(ids),
    onSuccess: () => {
      toast.success("Users restored")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-users"] })
      clearResourceSelection("user")
      setRestoringItemId(null)
    },
    onError: (error: unknown) => {
      toast.error("Failed to restore users", {
        description: getErrorDescription(error),
      })
      setRestoringItemId(null)
    },
  })

  const deleteUsersMutation = useMutation<unknown, AxiosError<{ error: string }>, string[]>({
    mutationFn: (ids) => recycleBinApi.permanentlyDeleteUsers(ids),
    onSuccess: () => {
      toast.success("Users permanently deleted")
      queryClient.invalidateQueries({ queryKey: ["recycle-bin-users"] })
      clearResourceSelection("user")
      setDeletingItemId(null)
    },
    onError: (error: unknown) => {
      toast.error("Failed to delete users", {
        description: getErrorDescription(error),
      })
      setDeletingItemId(null)
    },
  })

  const handleRestore = React.useCallback(() => {
    if (activeTab === "users" && selectedUserIds.length > 0) {
      restoreUsersMutation.mutate(selectedUserIds)
      return
    }
    if (activeTab === "projects" && selectedProjectIds.length > 0) {
      restoreProjectsMutation.mutate(selectedProjectIds)
      return
    }
    if (activeTab === "apps" && selectedAppIds.length > 0) {
      restoreAppsMutation.mutate(selectedAppIds)
      return
    }
    if (activeTab === "envs" && selectedEnvIds.length > 0) {
      restoreEnvsMutation.mutate(selectedEnvIds)
      return
    }
    if (activeTab === "code-repos" && selectedCodeRepoIds.length > 0) {
      restoreCodeReposMutation.mutate(selectedCodeRepoIds)
    }
  }, [
    activeTab,
    restoreAppsMutation,
    restoreCodeReposMutation,
    restoreEnvsMutation,
    restoreProjectsMutation,
    restoreUsersMutation,
    selectedAppIds,
    selectedCodeRepoIds,
    selectedEnvIds,
    selectedProjectIds,
    selectedUserIds,
  ])

  const handleDelete = React.useCallback(() => {
    if (activeTab === "users" && selectedUserIds.length > 0) {
      deleteUsersMutation.mutate(selectedUserIds)
      return
    }
    if (activeTab === "projects" && selectedProjectIds.length > 0) {
      deleteProjectsMutation.mutate(selectedProjectIds)
      return
    }
    if (activeTab === "apps" && selectedAppIds.length > 0) {
      deleteAppsMutation.mutate(selectedAppIds)
      return
    }
    if (activeTab === "envs" && selectedEnvIds.length > 0) {
      deleteEnvsMutation.mutate(selectedEnvIds)
      return
    }
    if (activeTab === "code-repos" && selectedCodeRepoIds.length > 0) {
      deleteCodeReposMutation.mutate(selectedCodeRepoIds)
    }
  }, [
    activeTab,
    deleteAppsMutation,
    deleteCodeReposMutation,
    deleteEnvsMutation,
    deleteProjectsMutation,
    deleteUsersMutation,
    selectedAppIds,
    selectedCodeRepoIds,
    selectedEnvIds,
    selectedProjectIds,
    selectedUserIds,
  ])

  const handleRestoreSingle = React.useCallback((id: string, type: RecycleBinResourceType) => {
    setRestoringItemId(id)
    if (type === "user") {
      restoreUsersMutation.mutate([id])
      return
    }
    if (type === "project") {
      restoreProjectsMutation.mutate([id])
      return
    }
    if (type === "app") {
      restoreAppsMutation.mutate([id])
      return
    }
    if (type === "env") {
      restoreEnvsMutation.mutate([id])
      return
    }
    restoreCodeReposMutation.mutate([id])
  }, [restoreAppsMutation, restoreCodeReposMutation, restoreEnvsMutation, restoreProjectsMutation, restoreUsersMutation])

  const handleDeleteSingle = React.useCallback((id: string, type: RecycleBinResourceType) => {
    setDeletingItemId(id)
    if (type === "user") {
      deleteUsersMutation.mutate([id])
      return
    }
    if (type === "project") {
      deleteProjectsMutation.mutate([id])
      return
    }
    if (type === "app") {
      deleteAppsMutation.mutate([id])
      return
    }
    if (type === "env") {
      deleteEnvsMutation.mutate([id])
      return
    }
    deleteCodeReposMutation.mutate([id])
  }, [deleteAppsMutation, deleteCodeReposMutation, deleteEnvsMutation, deleteProjectsMutation, deleteUsersMutation])

  const selectedCount = activeTab === "users"
    ? selectedUserIds.length
    : activeTab === "projects"
      ? selectedProjectIds.length
      : activeTab === "apps"
        ? selectedAppIds.length
        : activeTab === "envs"
          ? selectedEnvIds.length
          : selectedCodeRepoIds.length

  const selectedResourceLabel = activeTab === "users"
    ? "user(s)"
    : activeTab === "projects"
      ? "project(s)"
      : activeTab === "apps"
        ? "application(s)"
        : activeTab === "envs"
          ? "environment(s)"
          : "code repositor(ies)"

  return {
    searchQuery,
    setSearchQuery,
    restoringItemId,
    deletingItemId,
    conflictDialogOpen,
    setConflictDialogOpen,
    conflictApps,
    handleRestore,
    handleDelete,
    handleRestoreSingle,
    handleDeleteSingle,
    selectedCount,
    selectedResourceLabel,
    apps: {
      data: apps,
      isLoading: appsLoading,
      isFetching: appsFetching,
      refetch: refetchApps,
      pagination: appsPagination,
      setPagination: setAppsPagination,
      rowSelection: selectedAppRows,
      setRowSelection: setSelectedAppRows,
      selectedIds: selectedAppIds,
      paginationInfo: appsResponse?.pagination,
    } satisfies RecycleBinResourceState<RecycleBinApp>,
    envs: {
      data: envs,
      isLoading: envsLoading,
      isFetching: envsFetching,
      refetch: refetchEnvs,
      pagination: envsPagination,
      setPagination: setEnvsPagination,
      rowSelection: selectedEnvRows,
      setRowSelection: setSelectedEnvRows,
      selectedIds: selectedEnvIds,
      paginationInfo: envsResponse?.pagination,
    } satisfies RecycleBinResourceState<RecycleBinEnv>,
    projects: {
      data: projects,
      isLoading: projectsLoading,
      isFetching: projectsFetching,
      refetch: refetchProjects,
      pagination: projectsPagination,
      setPagination: setProjectsPagination,
      rowSelection: selectedProjectRows,
      setRowSelection: setSelectedProjectRows,
      selectedIds: selectedProjectIds,
      paginationInfo: projectsResponse?.pagination,
    } satisfies RecycleBinResourceState<RecycleBinProject>,
    codeRepos: {
      data: codeRepos,
      isLoading: codeReposLoading,
      isFetching: codeReposFetching,
      refetch: refetchCodeRepos,
      pagination: codeReposPagination,
      setPagination: setCodeReposPagination,
      rowSelection: selectedCodeRepoRows,
      setRowSelection: setSelectedCodeRepoRows,
      selectedIds: selectedCodeRepoIds,
      paginationInfo: codeReposResponse?.pagination,
    } satisfies RecycleBinResourceState<RecycleBinCodeRepo>,
    users: {
      data: users,
      isLoading: usersLoading,
      isFetching: usersFetching,
      refetch: refetchUsers,
      pagination: usersPagination,
      setPagination: setUsersPagination,
      rowSelection: selectedUserRows,
      setRowSelection: setSelectedUserRows,
      selectedIds: selectedUserIds,
      paginationInfo: usersResponse?.pagination,
    } satisfies RecycleBinResourceState<RecycleBinUser>,
  }
}
