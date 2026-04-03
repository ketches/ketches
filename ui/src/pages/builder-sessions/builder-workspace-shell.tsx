import {
  builderSessionsApi,
  type BuilderExport,
  type BuilderExportDeployBuildRequest,
  type BuilderExportInitialBuildPromotionRequest,
  type BuilderExportPromotionRequest,
  type BuilderMessage,
  type BuilderModelOption,
  type BuilderPreviewLaunch,
  type BuilderRun,
  type BuilderRunStatus,
  type BuilderSession,
  type BuilderSessionDetail,
  type BuilderWorkspaceFile,
} from "@/api/builder-sessions"
import { envsApi, type Env } from "@/api/envs"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import {
  Bot,
  Brain,
  ChevronLeft,
  ChevronRight,
  ChevronsUpDown,
  FileCode2,
  FileText,
  Folder,
  Loader2,
  MessageSquare,
  Orbit,
  Sparkles,
  User
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"
import { BuilderComposer } from "./builder-composer"
import { BuilderPreviewFrame } from "./builder-preview-frame"
import { BuilderPreviewPanel } from "./builder-preview-panel"
import { BuilderSessionHistoryRail } from "./builder-session-history-rail"

const AUTO_RESUME_WINDOW_MS = 8 * 60 * 60 * 1000

function sortSessionsByActivity(sessions: BuilderSession[]): BuilderSession[] {
  return [...sessions].sort((left, right) => {
    return new Date(right.last_activity_at).getTime() - new Date(left.last_activity_at).getTime()
  })
}

function getResumableSession(sessions: BuilderSession[], currentUserId: string): BuilderSession | null {
  const freshestOwnedSession = sortSessionsByActivity(
    sessions.filter((session) => session.created_by === currentUserId)
  )[0]

  if (!freshestOwnedSession) {
    return null
  }

  const sessionAgeMs = Date.now() - new Date(freshestOwnedSession.last_activity_at).getTime()
  return sessionAgeMs < AUTO_RESUME_WINDOW_MS ? freshestOwnedSession : null
}

function isRunActive(status: BuilderRunStatus | undefined): boolean {
  return status === "queued" || status === "executing"
}

function isSessionAppendable(status: BuilderSession["status"] | string | null | undefined): boolean {
  return status === "provisioning" || status === "ready" || status === "running"
}

function isBuilderSessionNotAppendableError(error: unknown): boolean {
  if (!isAxiosError(error)) {
    return false
  }

  const message = error.response?.data?.error
  return error.response?.status === 409 && typeof message === "string" && message.includes("is not appendable")
}

function getMessageKey(message: BuilderMessage): string {
  return message.id || `${message.role}-${message.created_at}`
}

function getSelectedBuildEnvironment(
  session: BuilderSession | null,
  draftBuildEnvId: string,
  buildEnvs: Env[]
): Env | null {
  if (session?.build_env_id) {
    return buildEnvs.find((env) => env.id === session.build_env_id) ?? null
  }

  if (draftBuildEnvId) {
    return buildEnvs.find((env) => env.id === draftBuildEnvId) ?? null
  }

  return null
}

const HIDDEN_SYSTEM_MESSAGE_CONTENT = "run completed: replied without workspace changes"

interface BuilderQueuedMessageItem {
  id: string
  content: string
  createdAt: string
  optimistic?: boolean
}

function isHiddenSystemMessage(message: BuilderMessage): boolean {
  return message.role === "system" && message.content.trim() === HIDDEN_SYSTEM_MESSAGE_CONTENT
}

function getRunById(runs: BuilderRun[]): Map<string, BuilderRun> {
  return new Map(runs.map((run) => [run.id, run]))
}

function getQueuedConversationMessages(messages: BuilderMessage[], runs: BuilderRun[]): BuilderMessage[] {
  const runById = getRunById(runs)

  return messages.filter((message) => {
    if (message.role !== "user") {
      return false
    }

    return runById.get(message.run_id)?.status === "queued"
  })
}

function getVisibleConversationMessages(messages: BuilderMessage[], runs: BuilderRun[]): BuilderMessage[] {
  const queuedMessages = new Set(getQueuedConversationMessages(messages, runs).map((message) => message.id))

  return messages.filter((message) => {
    if (queuedMessages.has(message.id)) {
      return false
    }

    return !isHiddenSystemMessage(message)
  })
}

function getActiveStreamingRun(runs: BuilderRun[]): BuilderRun | null {
  return runs.find((run) => run.status === "executing") ?? runs.find((run) => run.status === "queued") ?? null
}

function mergeQueuedMessageItems(
  optimisticMessages: BuilderQueuedMessageItem[],
  persistedMessages: BuilderQueuedMessageItem[]
): BuilderQueuedMessageItem[] {
  const merged = [...persistedMessages, ...optimisticMessages]
  const seen = new Set<string>()

  return merged
    .filter((message) => {
      const key = `${message.content}-${message.createdAt}`
      if (seen.has(key)) {
        return false
      }
      seen.add(key)
      return true
    })
    .sort((left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime())
}

function mergeBuilderSessionDetailAfterPostMessage(
  detail: BuilderSessionDetail | undefined,
  payload: {
    session: BuilderSession
    message: BuilderMessage
    run: BuilderRun
  }
): BuilderSessionDetail | undefined {
  if (!detail) {
    return detail
  }

  const nextMessages = detail.messages.some((message) => message.id === payload.message.id)
    ? detail.messages.map((message) => (message.id === payload.message.id ? payload.message : message))
    : [...detail.messages, payload.message]
  const nextRuns = detail.runs.some((run) => run.id === payload.run.id)
    ? detail.runs.map((run) => (run.id === payload.run.id ? payload.run : run))
    : [payload.run, ...detail.runs]

  return {
    ...detail,
    session: payload.session,
    messages: nextMessages,
    runs: nextRuns,
  }
}

export function BuilderWorkspaceShell() {
  const { projectId, sessionId } = useParams<{ projectId: string; sessionId?: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const currentUserId = useAuthStore((state) => state.user?.id ?? "")
  const activeProjectName = useProjectStore((state) => state.activeProjectName)
  const activeEnvName = useProjectStore((state) => state.activeEnvName)
  const setActiveContextWithNames = useProjectStore((state) => state.setActiveContextWithNames)

  const [messageInput, setMessageInput] = React.useState("")
  const [draftBuildEnvId, setDraftBuildEnvId] = React.useState("")
  const [draftModelKey, setDraftModelKey] = React.useState<string | null>(null)
  const [draftError, setDraftError] = React.useState("")
  const [filesExpanded, setFilesExpanded] = React.useState(false)
  const [currentPath, setCurrentPath] = React.useState("/")
  const [selectedFile, setSelectedFile] = React.useState<BuilderWorkspaceFile | null>(null)
  const [fileContent, setFileContent] = React.useState<string | null>(null)
  const [previewLaunch, setPreviewLaunch] = React.useState<BuilderPreviewLaunch | null>(null)
  const [promotingExportId, setPromotingExportId] = React.useState<string | null>(null)
  const [promotionForm, setPromotionForm] = React.useState<BuilderExportPromotionRequest>({
    name: "",
    slug: "",
    git_repo_url: "",
    git_username: "",
    git_password: "",
  })
  const [buildPromotionForm, setBuildPromotionForm] = React.useState<BuilderExportInitialBuildPromotionRequest>({
    name: "",
    slug: "",
    git_repo_url: "",
    git_username: "",
    git_password: "",
    build_env_id: "",
    registry_id: "",
    build_setting_name: "",
    image_name: "",
    dockerfile_path: "",
    build_context: "",
    git_ref: "",
  })
  const [deployBuildForm, setDeployBuildForm] = React.useState<BuilderExportDeployBuildRequest>({
    repository_id: "",
    build_id: "",
    target_env_id: "",
    app_id: "",
    name: "Builder App",
    slug: "builder-app",
  })
  const [optimisticQueuedMessages, setOptimisticQueuedMessages] = React.useState<BuilderQueuedMessageItem[]>([])
  const [streamingRunId, setStreamingRunId] = React.useState<string | null>(null)
  const [streamingLog, setStreamingLog] = React.useState("")
  const composerRef = React.useRef<HTMLTextAreaElement | null>(null)

  const { data: sessionsResponse, isLoading: isSessionsLoading } = useQuery({
    queryKey: ["builder-sessions", projectId],
    queryFn: () => builderSessionsApi.list(projectId!),
    enabled: !!projectId,
  })

  const sessions = React.useMemo(
    () => sortSessionsByActivity(sessionsResponse?.items ?? []),
    [sessionsResponse?.items]
  )

  const resumableSession = React.useMemo(
    () => getResumableSession(sessions, currentUserId),
    [currentUserId, sessions]
  )
  const isDraftOverride = searchParams.get("draft") === "1"

  React.useEffect(() => {
    if (!projectId || sessionId || isSessionsLoading || !resumableSession || isDraftOverride) {
      return
    }

    navigate(`/projects/${projectId}/builder-sessions/${resumableSession.id}`, { replace: true })
  }, [isDraftOverride, isSessionsLoading, navigate, projectId, resumableSession, sessionId])

  const { data: buildEnvs = [] } = useQuery({
    queryKey: ["builder-build-envs", projectId],
    queryFn: async () => {
      const response = await envsApi.list(projectId!, { page: 1, page_size: 100 })
      return (response.items ?? []).filter((env: Env) => env.is_build_env)
    },
    enabled: !!projectId,
  })

  const draftBuildEnvOptions = React.useMemo(
    () => buildEnvs.map((env) => ({ label: env.name, value: env.id })),
    [buildEnvs]
  )
  const { data: modelSelection } = useQuery({
    queryKey: ["builder-model-selection", projectId],
    queryFn: () => builderSessionsApi.getModelSelection(projectId!),
    enabled: !!projectId,
  })
  const draftModelOptions = React.useMemo<BuilderModelOption[]>(
    () => modelSelection?.options ?? [],
    [modelSelection?.options]
  )

  React.useEffect(() => {
    if (sessionId || draftBuildEnvId || draftBuildEnvOptions.length !== 1) {
      return
    }

    setDraftBuildEnvId(draftBuildEnvOptions[0].value)
  }, [draftBuildEnvId, draftBuildEnvOptions, sessionId])

  React.useEffect(() => {
    if (draftModelKey || !modelSelection) {
      return
    }

    if (modelSelection.effectiveDefaultOption) {
      setDraftModelKey(modelSelection.effectiveDefaultOption.key)
    }
  }, [draftModelKey, modelSelection])

  const modelSelectionHint = React.useMemo(() => {
    if (!modelSelection) {
      return ""
    }

    if (modelSelection.effectiveDefaultSource === "project") {
      if (draftModelKey && modelSelection.effectiveDefaultOption && draftModelKey !== modelSelection.effectiveDefaultOption.key) {
        return "Overrides project default"
      }

      return "Default from project settings"
    }

    if (modelSelection.effectiveDefaultSource === "user") {
      return "Default from your account settings"
    }

    return ""
  }, [draftModelKey, modelSelection])

  const { data: sessionDetail, isLoading: isSessionLoading, error: sessionError } = useQuery({
    queryKey: ["builder-session", projectId, sessionId],
    queryFn: () => builderSessionsApi.get(projectId!, sessionId!),
    enabled: !!projectId && !!sessionId,
    refetchInterval: ({ state }) => {
      const detail = state.data as BuilderSessionDetail | undefined
      return isRunActive(detail?.session.latest_run_status) ? 3000 : false
    },
  })

  const selectedDetail: BuilderSessionDetail | undefined = sessionDetail
  const selectedSession = selectedDetail?.session ?? null
  const selectedPreview = selectedDetail?.preview
  const selectedPreviewRunId = selectedPreview?.resolved_run_id ?? null
  const selectedPreviewAvailable = selectedPreview?.preview_available ?? false
  const selectedRuns = selectedDetail?.runs ?? []
  const rawMessages = selectedDetail?.messages ?? []
  const latestRun = React.useMemo(() => {
    if (!selectedDetail?.runs?.length || !selectedDetail.session.latest_run_id) {
      return selectedDetail?.runs?.[0]
    }

    return selectedDetail.runs.find((run) => run.id === selectedDetail.session.latest_run_id) ?? selectedDetail.runs[0]
  }, [selectedDetail])
  const queuedConversationMessages = React.useMemo(
    () => getQueuedConversationMessages(rawMessages, selectedRuns),
    [rawMessages, selectedRuns]
  )
  const visibleMessages = React.useMemo(
    () => getVisibleConversationMessages(rawMessages, selectedRuns),
    [rawMessages, selectedRuns]
  )
  const queuedMessages = React.useMemo(
    () =>
      mergeQueuedMessageItems(
        optimisticQueuedMessages,
        queuedConversationMessages.map((message) => ({
          id: message.id,
          content: message.content,
          createdAt: message.created_at,
        }))
      ),
    [optimisticQueuedMessages, queuedConversationMessages]
  )
  const activeStreamingRun = React.useMemo(
    () => getActiveStreamingRun(selectedRuns),
    [selectedRuns]
  )
  const isAnyRunActive = React.useMemo(
    () => selectedRuns.some((run) => isRunActive(run.status)),
    [selectedRuns]
  )
  const hasFiles =
    !!selectedDetail &&
    (((selectedDetail.artifacts?.length ?? 0) > 0) || ((selectedDetail.session.artifact_count ?? 0) > 0))

  const { data: filesData } = useQuery({
    queryKey: ["builder-files", projectId, sessionId, currentPath],
    queryFn: () => builderSessionsApi.listFiles(projectId!, sessionId!, currentPath),
    enabled: !!projectId && !!sessionId && hasFiles,
  })
  const { data: sessionExports = [] } = useQuery({
    queryKey: ["builder-exports", projectId, sessionId],
    queryFn: () => builderSessionsApi.listExports(projectId!, sessionId!),
    enabled: !!projectId && !!sessionId,
  })
  const { data: exportPromotionPlan } = useQuery({
    queryKey: ["builder-export-promotion-plan", projectId, sessionId, promotingExportId],
    queryFn: () => builderSessionsApi.getExportPromotionPlan(projectId!, sessionId!, promotingExportId!),
    enabled: !!projectId && !!sessionId && !!promotingExportId,
  })

  React.useEffect(() => {
    setFilesExpanded(false)
    setCurrentPath("/")
    setSelectedFile(null)
    setFileContent(null)
    setPreviewLaunch(null)
    setOptimisticQueuedMessages([])
    setStreamingRunId(null)
    setStreamingLog("")
  }, [sessionId])

  const createSessionMutation = useMutation({
    mutationFn: (payload: { buildEnvId: string; prompt: string; modelKey: string | null }) => {
      const selectedModelOption = draftModelOptions.find((option) => option.key === payload.modelKey)

      return builderSessionsApi.create(projectId!, {
        build_env_id: payload.buildEnvId,
        prompt: payload.prompt,
        selected_model_key: selectedModelOption?.key,
        provider_key: selectedModelOption?.providerKey,
        model_profile_key: selectedModelOption?.modelProfileKey,
      })
    },
    onSuccess: (detail: BuilderSessionDetail) => {
      setMessageInput("")
      setDraftError("")
      queryClient.invalidateQueries({ queryKey: ["builder-sessions", projectId] })
      navigate(`/projects/${projectId}/builder-sessions/${detail.session.id}`)
    },
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to create Builder session", { description: message })
    },
  })

  const startReplacementSession = React.useCallback(
    (content: string, modelKey: string | null, buildEnvId: string) => {
      createSessionMutation.mutate({ buildEnvId, prompt: content, modelKey })
    },
    [createSessionMutation]
  )

  const sendMessageMutation = useMutation({
    mutationFn: (payload: {
      selectedSessionId: string
      content: string
      modelKey: string | null
    }) => {
      const selectedModelOption = draftModelOptions.find((option) => option.key === payload.modelKey)

      return builderSessionsApi.postMessage(projectId!, payload.selectedSessionId, {
        content: payload.content,
        selected_model_key: selectedModelOption?.key,
        provider_key: selectedModelOption?.providerKey,
        model_profile_key: selectedModelOption?.modelProfileKey,
      })
    },
    onMutate: (variables) => {
      const shouldQueue = isAnyRunActive || isRunActive(selectedSession?.latest_run_status)
      if (!shouldQueue) {
        setMessageInput("")
        return { optimisticQueuedMessageId: null as string | null }
      }

      const optimisticQueuedMessageId = `queued-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
      const now = new Date().toISOString()

      setOptimisticQueuedMessages((current) => [
        ...current,
        {
          id: optimisticQueuedMessageId,
          content: variables.content,
          createdAt: now,
          optimistic: true,
        },
      ])
      setMessageInput("")

      return { optimisticQueuedMessageId }
    },
    onSuccess: (payload, variables, context) => {
      if (context?.optimisticQueuedMessageId) {
        setOptimisticQueuedMessages((current) =>
          current.map((message) =>
            message.id === context.optimisticQueuedMessageId
              ? {
                ...message,
                id: payload.message.id,
                createdAt: payload.message.created_at,
              }
              : message
          )
        )
      }

      queryClient.setQueryData<BuilderSessionDetail | undefined>(
        ["builder-session", projectId, variables.selectedSessionId],
        (current) => mergeBuilderSessionDetailAfterPostMessage(current, payload)
      )
      queryClient.invalidateQueries({ queryKey: ["builder-session", projectId, sessionId] })
      queryClient.invalidateQueries({ queryKey: ["builder-sessions", projectId] })
    },
    onError: (error: unknown, variables, context) => {
      if (context?.optimisticQueuedMessageId) {
        setOptimisticQueuedMessages((current) =>
          current.filter((message) => message.id !== context.optimisticQueuedMessageId)
        )
      }

      if (isBuilderSessionNotAppendableError(error) && selectedSession?.build_env_id) {
        startReplacementSession(variables.content, variables.modelKey, selectedSession.build_env_id)
        return
      }

      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to send message", { description: message })
    },
  })
  const createExportMutation = useMutation({
    mutationFn: () => builderSessionsApi.createExport(projectId!, sessionId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["builder-exports", projectId, sessionId] })
    },
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to create export", { description: message })
    },
  })
  const promoteExportMutation = useMutation({
    mutationFn: ({ exportId, payload }: { exportId: string; payload: BuilderExportPromotionRequest }) =>
      builderSessionsApi.promoteExportToRepository(projectId!, sessionId!, exportId, payload),
    onSuccess: () => {
      setPromotingExportId(null)
      setPromotionForm({
        name: "",
        slug: "",
        git_repo_url: "",
        git_username: "",
        git_password: "",
      })
      queryClient.invalidateQueries({ queryKey: ["builder-exports", projectId, sessionId] })
    },
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to promote export", { description: message })
    },
  })
  const promoteExportToBuildMutation = useMutation({
    mutationFn: ({ exportId, payload }: { exportId: string; payload: BuilderExportInitialBuildPromotionRequest }) =>
      builderSessionsApi.promoteExportToInitialBuild(projectId!, sessionId!, exportId, payload),
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to promote export to initial build", { description: message })
    },
  })
  const deployExportBuildMutation = useMutation({
    mutationFn: (payload: BuilderExportDeployBuildRequest) =>
      builderSessionsApi.deployExportBuild(projectId!, sessionId!, promotingExportId!, payload),
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to deploy build", { description: message })
    },
  })

  React.useEffect(() => {
    if (!exportPromotionPlan) {
      return
    }

    setBuildPromotionForm((current) => ({
      ...current,
      build_env_id: exportPromotionPlan.suggested_build_env_id || current.build_env_id,
      build_setting_name: exportPromotionPlan.suggested_build_setting_name || current.build_setting_name,
      image_name: exportPromotionPlan.suggested_image_name || current.image_name,
      dockerfile_path: exportPromotionPlan.suggested_dockerfile_path || current.dockerfile_path,
      build_context: exportPromotionPlan.suggested_build_context || current.build_context,
      git_ref: current.git_ref || "main",
    }))
  }, [exportPromotionPlan])

  React.useEffect(() => {
    if (!promoteExportToBuildMutation.data || !exportPromotionPlan) {
      return
    }

    setDeployBuildForm((current) => ({
      ...current,
      repository_id: promoteExportToBuildMutation.data.promotion.repository.id,
      build_id: promoteExportToBuildMutation.data.build.id,
      target_env_id: exportPromotionPlan.suggested_build_env_id || current.target_env_id,
    }))
  }, [exportPromotionPlan, promoteExportToBuildMutation.data])

  React.useEffect(() => {
    setOptimisticQueuedMessages((current) =>
      {
        const next = current.filter((message) => {
        const existsInQueuedMessages = queuedConversationMessages.some((queuedMessage) => queuedMessage.id === message.id)
        if (existsInQueuedMessages) {
          return false
        }

        return isAnyRunActive
        })

        if (next.length === current.length && next.every((message, index) => message === current[index])) {
          return current
        }

        return next
      }
    )
  }, [isAnyRunActive, queuedConversationMessages])

  React.useEffect(() => {
    if (!projectId || !sessionId || !activeStreamingRun || !isRunActive(activeStreamingRun.status)) {
      setStreamingRunId(null)
      setStreamingLog("")
      return
    }

    const streamUrl = builderSessionsApi.runLogsStreamUrl(projectId, sessionId, activeStreamingRun.id)
    const eventSource = new EventSource(streamUrl, { withCredentials: true })

    setStreamingRunId(activeStreamingRun.id)
    setStreamingLog("")

    eventSource.addEventListener("log", (event: MessageEvent<string> | { data: string }) => {
      const chunk = typeof event.data === "string" ? event.data : ""
      if (!chunk) {
        return
      }

      React.startTransition(() => {
        setStreamingLog((current) => current + chunk)
      })
    })

    eventSource.addEventListener("done", () => {
      eventSource.close()
    })

    eventSource.onerror = () => {
      eventSource.close()
    }

    return () => {
      eventSource.close()
    }
  }, [activeStreamingRun?.id, activeStreamingRun?.status, projectId, sessionId])

  const isSubmitting = createSessionMutation.isPending
  const isRedirectingToFreshSession = !sessionId && !isSessionsLoading && !!resumableSession && !isDraftOverride

  const handleComposerInputChange = React.useCallback((value: string) => {
    setMessageInput(value)
  }, [])

  const handleSendMessage = React.useCallback(() => {
    const content = (messageInput || composerRef.current?.value || "").trim()
    if (!content) {
      return
    }

    if (sessionId) {
      if (selectedSession && !isSessionAppendable(selectedSession.status)) {
        if (!selectedSession.build_env_id) {
          toast.error("Failed to create Builder session", { description: "Build environment is required" })
          return
        }

        startReplacementSession(content, draftModelKey, selectedSession.build_env_id)
        return
      }

      sendMessageMutation.mutate({ selectedSessionId: sessionId, content, modelKey: draftModelKey })
      return
    }

    if (!draftBuildEnvId) {
      setDraftError("Build environment is required")
      return
    }

    createSessionMutation.mutate({ buildEnvId: draftBuildEnvId, prompt: content, modelKey: draftModelKey })
  }, [createSessionMutation, draftBuildEnvId, draftModelKey, messageInput, selectedSession, sendMessageMutation, sessionId, startReplacementSession])

  const handleSelectFile = React.useCallback(
    async (file: BuilderWorkspaceFile) => {
      if (!projectId || !sessionId) {
        return
      }

      if (file.type === "dir") {
        setSelectedFile(null)
        setFileContent(null)
        const nextPath = currentPath === "/" ? `/${file.name}/` : `${currentPath}${file.name}/`
        setCurrentPath(nextPath)
        return
      }

      setSelectedFile(file)
      setFileContent(null)

      try {
        const response = await builderSessionsApi.readFile(projectId, sessionId, `${currentPath}${file.name}`)
        setFileContent(response.content)
      } catch {
        setFileContent("Failed to load file preview.")
      }
    },
    [currentPath, projectId, sessionId]
  )

  const handleOpenDraft = React.useCallback(() => {
    if (!projectId) {
      return
    }

    navigate(`/projects/${projectId}/builder-sessions?draft=1`)
  }, [navigate, projectId])

  const handleDownloadFiles = React.useCallback(async () => {
    if (!projectId || !sessionId) {
      return
    }

    try {
      await builderSessionsApi.downloadTarBlob(projectId, sessionId)
    } catch {
      toast.error("Failed to download files")
    }
  }, [projectId, sessionId])

  const handleDownloadPreview = React.useCallback(async () => {
    if (!projectId || !sessionId || !selectedPreviewRunId) {
      return
    }

    try {
      await builderSessionsApi.downloadPreviewSnapshotBlob(projectId, sessionId, selectedPreviewRunId)
    } catch {
      toast.error("Failed to download preview snapshot")
    }
  }, [projectId, selectedPreviewRunId, sessionId])

  const handleOpenPreview = React.useCallback(async () => {
    if (!projectId || !sessionId || !selectedPreviewRunId || !selectedPreviewAvailable) {
      return
    }

    try {
      const launch = await builderSessionsApi.launchPreview(projectId, sessionId, selectedPreviewRunId)
      setPreviewLaunch(launch)
    } catch {
      toast.error("Failed to open preview")
    }
  }, [projectId, selectedPreviewAvailable, selectedPreviewRunId, sessionId])

  const handleDownloadExport = React.useCallback(async (exportItem: BuilderExport) => {
    if (!projectId || !sessionId) {
      return
    }

    try {
      await builderSessionsApi.downloadExportBlob(projectId, sessionId, exportItem.id)
    } catch {
      toast.error("Failed to download export")
    }
  }, [projectId, sessionId])

  const handlePromoteExport = React.useCallback((exportId: string) => {
    promoteExportMutation.mutate({
      exportId,
      payload: promotionForm,
    })
  }, [promoteExportMutation, promotionForm])

  const handlePromoteExportToBuild = React.useCallback((exportId: string) => {
    promoteExportToBuildMutation.mutate({
      exportId,
      payload: buildPromotionForm,
    })
  }, [buildPromotionForm, promoteExportToBuildMutation])

  const handleDeployExportBuild = React.useCallback(() => {
    deployExportBuildMutation.mutate(deployBuildForm)
  }, [deployBuildForm, deployExportBuildMutation])

  const selectedBuildEnvironment = getSelectedBuildEnvironment(selectedSession, draftBuildEnvId, buildEnvs)
  const environmentLabel = selectedBuildEnvironment?.name || activeEnvName || "Environment"
  const breadcrumbItems = React.useMemo<BreadcrumbItem[]>(() => {
    const environmentDropdown = buildEnvs.length > 0 ? (
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
        <DropdownMenuContent align="start">
          <DropdownMenuGroup>
            {buildEnvs.map((env) => (
              <DropdownMenuItem
                key={env.id}
                onClick={() => {
                  setActiveContextWithNames(projectId ?? null, activeProjectName ?? null, env.id, env.name)
                  setDraftBuildEnvId(env.id)
                  setDraftError("")

                  if (!projectId) {
                    return
                  }

                  if (sessionId) {
                    navigate(`/projects/${projectId}/builder-sessions?draft=1`)
                    return
                  }

                  navigate(`/projects/${projectId}/builder-sessions?draft=1`, { replace: !isDraftOverride })
                }}
              >
                <Orbit className="h-4 w-4" />
                {env.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    ) : undefined

    return [
      { label: "Builder", href: `/projects/${projectId}/builder-sessions`, icon: Sparkles },
      { label: environmentLabel, icon: Orbit, dropdown: environmentDropdown },
      { label: "Current Session", icon: MessageSquare },
    ]
  }, [
    activeProjectName,
    buildEnvs,
    environmentLabel,
    isDraftOverride,
    navigate,
    projectId,
    sessionId,
    setActiveContextWithNames,
  ])
  const shouldShowConversationLoader = (isSessionsLoading && sessions.length === 0) || isRedirectingToFreshSession
  const shouldShowSessionError = !!sessionId && !!sessionError
  const composerStatusText = selectedBuildEnvironment
    ? undefined
    : buildEnvs.length > 0
      ? "Select a build environment before your first send"
      : "No build environments available"

  return (
    <div
      data-testid="builder-workspace-shell"
      className="flex h-full min-h-0 flex-1 flex-col overflow-hidden"
    >
      <PageHeader items={breadcrumbItems} />

      <div
        data-testid="builder-workspace-body"
        className="flex h-full min-h-0 flex-1 overflow-hidden bg-background"
      >
        <BuilderSessionHistoryRail
          sessions={sessions}
          selectedSessionId={sessionId}
          onNewConversation={handleOpenDraft}
          onSelectSession={(targetSessionId) => {
            if (!projectId) {
              return
            }
            navigate(`/projects/${projectId}/builder-sessions/${targetSessionId}`)
          }}
        />

        <section
          data-testid="builder-workspace-chat-column"
          className="flex h-full min-h-0 min-w-0 flex-1 overflow-hidden bg-muted/5"
        >
          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="flex min-h-0 flex-1 flex-col">
              <div className={`min-h-0 flex-1 overflow-y-auto px-5 py-5 ${sessionId ? "" : "flex items-center justify-center"}`}>
                {shouldShowConversationLoader ? (
                  <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Opening Builder workspace…
                  </div>
                ) : shouldShowSessionError ? (
                  <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                    Failed to load this Builder session.
                  </div>
                ) : sessionId && isSessionLoading && !selectedDetail ? (
                  <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Loading conversation…
                  </div>
                ) : sessionId ? (
                  <div className="mx-auto flex w-full max-w-4xl flex-col gap-4">
                    {latestRun?.executor_policy_key || latestRun?.execution_image_ref ? (
                      <div className="rounded-2xl border bg-background px-4 py-3 text-sm text-muted-foreground shadow-sm">
                        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                          {latestRun.planned_project_kind ? (
                            <span>Project kind: {latestRun.planned_project_kind}</span>
                          ) : null}
                          {latestRun.phase ? (
                            <span>Phase: {latestRun.phase}</span>
                          ) : null}
                          {latestRun.executor_policy_key ? (
                            <span>Executor: {latestRun.executor_policy_key}</span>
                          ) : null}
                          {latestRun.execution_image_ref ? (
                            <span>Image: {latestRun.execution_image_ref}</span>
                          ) : null}
                          {latestRun.error_class ? (
                            <span>Error class: {latestRun.error_class}</span>
                          ) : null}
                        </div>
                      </div>
                    ) : null}
                    <div className="rounded-2xl border bg-background px-4 py-3 text-sm text-muted-foreground shadow-sm">
                      <div className="mb-2 flex items-center justify-between gap-3">
                        <span className="font-medium text-foreground">Exports</span>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => createExportMutation.mutate()}
                          disabled={createExportMutation.isPending}
                        >
                          Create export
                        </Button>
                      </div>
                      {sessionExports.length === 0 ? (
                        <div>No exports yet.</div>
                      ) : (
                        <div className="space-y-2">
                          {sessionExports.map((exportItem) => (
                            <div key={exportItem.id} className="space-y-2 rounded-lg border px-3 py-2">
                              <div className="flex items-center justify-between gap-3">
                                <div className="min-w-0">
                                  <div className="truncate text-foreground">{exportItem.file_name}</div>
                                  <div className="text-xs text-muted-foreground">{exportItem.kind}</div>
                                </div>
                                <div className="flex items-center gap-2">
                                  <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() => void handleDownloadExport(exportItem)}
                                  >
                                    Download export
                                  </Button>
                                  <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setPromotingExportId((current) => current === exportItem.id ? null : exportItem.id)}
                                  >
                                    Promote to repository
                                  </Button>
                                </div>
                              </div>
                              {promotingExportId === exportItem.id ? (
                                <div className="space-y-2 border-t pt-2">
                                  <Input
                                    name="builder_export_name"
                                    placeholder="Repository name"
                                    value={promotionForm.name ?? ""}
                                    onInput={(event) => setPromotionForm((current) => ({ ...current, name: (event.target as HTMLInputElement).value }))}
                                  />
                                  <Input
                                    name="builder_export_slug"
                                    placeholder="Repository slug"
                                    value={promotionForm.slug ?? ""}
                                    onInput={(event) => setPromotionForm((current) => ({ ...current, slug: (event.target as HTMLInputElement).value }))}
                                  />
                                  <Input
                                    name="builder_export_git_repo_url"
                                    placeholder="Git repository URL"
                                    value={promotionForm.git_repo_url}
                                    onInput={(event) => {
                                      const value = (event.target as HTMLInputElement).value
                                      setPromotionForm((current) => ({ ...current, git_repo_url: value }))
                                      setBuildPromotionForm((current) => ({ ...current, git_repo_url: value }))
                                    }}
                                  />
                                  <Input
                                    name="builder_export_git_username"
                                    placeholder="Git username"
                                    value={promotionForm.git_username ?? ""}
                                    onInput={(event) => {
                                      const value = (event.target as HTMLInputElement).value
                                      setPromotionForm((current) => ({ ...current, git_username: value }))
                                      setBuildPromotionForm((current) => ({ ...current, git_username: value }))
                                    }}
                                  />
                                  <Input
                                    name="builder_export_git_password"
                                    placeholder="Git password"
                                    value={promotionForm.git_password ?? ""}
                                    onInput={(event) => {
                                      const value = (event.target as HTMLInputElement).value
                                      setPromotionForm((current) => ({ ...current, git_password: value }))
                                      setBuildPromotionForm((current) => ({ ...current, git_password: value }))
                                    }}
                                  />
                                  {exportPromotionPlan ? (
                                    <div className="space-y-2 rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                                      <div>Plan: {exportPromotionPlan.planned_project_kind}</div>
                                      <div>Suggested env: {exportPromotionPlan.suggested_build_env_id}</div>
                                      <div>Suggested image: {exportPromotionPlan.suggested_image_name}</div>
                                      <div>Can trigger initial build: {exportPromotionPlan.can_trigger_initial_build ? "yes" : "no"}</div>
                                      {exportPromotionPlan.missing_requirements.length > 0 ? (
                                        <div>{exportPromotionPlan.missing_requirements.join(", ")}</div>
                                      ) : null}
                                    </div>
                                  ) : null}
                                  <Input
                                    name="builder_export_registry_id"
                                    placeholder="Container registry ID"
                                    value={buildPromotionForm.registry_id}
                                    onInput={(event) => setBuildPromotionForm((current) => ({ ...current, registry_id: (event.target as HTMLInputElement).value }))}
                                  />
                                  {promoteExportToBuildMutation.data ? (
                                    <div className="space-y-2 rounded-lg border bg-muted/20 px-3 py-2">
                                      <div className="text-xs text-muted-foreground">Initial build ready: {promoteExportToBuildMutation.data.build.id}</div>
                                      <Input
                                        name="builder_export_deploy_target_env_id"
                                        placeholder="Target env ID"
                                        value={deployBuildForm.target_env_id}
                                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, target_env_id: (event.target as HTMLInputElement).value }))}
                                      />
                                      <Input
                                        name="builder_export_deploy_name"
                                        placeholder="App name"
                                        value={deployBuildForm.name ?? ""}
                                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, name: (event.target as HTMLInputElement).value }))}
                                      />
                                      <Input
                                        name="builder_export_deploy_slug"
                                        placeholder="App slug"
                                        value={deployBuildForm.slug ?? ""}
                                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, slug: (event.target as HTMLInputElement).value }))}
                                      />
                                    </div>
                                  ) : null}
                                  <div className="flex justify-end">
                                    <div className="flex gap-2">
                                      <Button
                                        type="button"
                                        size="sm"
                                        onClick={() => handlePromoteExport(exportItem.id)}
                                        disabled={promoteExportMutation.isPending || !promotionForm.git_repo_url}
                                      >
                                        Promote export
                                      </Button>
                                      <Button
                                        type="button"
                                        size="sm"
                                        onClick={() => handlePromoteExportToBuild(exportItem.id)}
                                        disabled={
                                          promoteExportToBuildMutation.isPending ||
                                          !buildPromotionForm.git_repo_url ||
                                          !buildPromotionForm.registry_id ||
                                          !buildPromotionForm.build_env_id
                                        }
                                      >
                                        Promote to initial build
                                      </Button>
                                      {promoteExportToBuildMutation.data ? (
                                        <Button
                                          type="button"
                                          size="sm"
                                          onClick={handleDeployExportBuild}
                                          disabled={
                                            deployExportBuildMutation.isPending ||
                                            !deployBuildForm.repository_id ||
                                            !deployBuildForm.build_id ||
                                            !deployBuildForm.target_env_id
                                          }
                                        >
                                          Deploy build
                                        </Button>
                                      ) : null}
                                    </div>
                                  </div>
                                </div>
                              ) : null}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                    {selectedPreview ? (
                      <BuilderPreviewPanel
                        preview={selectedPreview}
                        onDownload={handleDownloadPreview}
                        onOpenPreview={handleOpenPreview}
                      />
                    ) : null}
                    {previewLaunch?.frame_url ? <BuilderPreviewFrame frameUrl={previewLaunch.frame_url} /> : null}
                    {visibleMessages.length === 0 && !activeStreamingRun ? (
                      <div className="flex min-h-36 flex-col items-center justify-center rounded-[1.75rem] border bg-background px-7 py-8 text-center text-sm text-muted-foreground shadow-sm">
                        <Bot className="mb-2.5 h-5 w-5" />
                        No messages yet. Send the next instruction to keep building.
                      </div>
                    ) : (
                      <>
                        {visibleMessages.map((message) => <ConversationMessage key={getMessageKey(message)} message={message} />)}
                        {activeStreamingRun ? (
                          <StreamingConversationMessage
                            run={activeStreamingRun}
                            streamingLog={streamingRunId === activeStreamingRun.id ? streamingLog : ""}
                          />
                        ) : null}
                      </>
                    )}
                  </div>
                ) : (
                  <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
                    <EmptyState
                      title="New conversation"
                      description="Describe what you want to build. The draft stays unsaved until your first send."
                      icon={Brain}
                    />

                    <div className="mx-auto w-full max-w-3xl">
                      <BuilderComposer
                        centered
                        composerRef={composerRef}
                        value={messageInput}
                        onValueChange={handleComposerInputChange}
                        onSubmit={handleSendMessage}
                        isSubmitting={isSubmitting}
                        modelValue={draftModelKey}
                        modelOptions={draftModelOptions}
                        onModelValueChange={setDraftModelKey}
                        modelSelectionHint={modelSelectionHint}
                        statusText={composerStatusText}
                        statusError={draftError || undefined}
                      />
                    </div>
                  </div>

                )}
              </div>

              {sessionId ? (
                <div className="shrink-0 space-y-3">
                  {queuedMessages.length > 0 ? (
                    <div
                      data-testid="builder-queued-messages"
                      className="mx-4 shrink-0 rounded-2xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm shadow-sm"
                    >
                      <div className="mb-2 font-medium text-amber-950">Queued next</div>
                      <div className="space-y-2">
                        {queuedMessages.map((message) => (
                          <div
                            key={message.id}
                            className="rounded-xl border border-amber-200/80 bg-background/80 px-3 py-2 text-foreground"
                          >
                            {message.content}
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  <BuilderComposer
                    composerRef={composerRef}
                    value={messageInput}
                    onValueChange={handleComposerInputChange}
                    onSubmit={handleSendMessage}
                    isSubmitting={isSubmitting}
                    modelValue={draftModelKey}
                    modelOptions={draftModelOptions}
                    onModelValueChange={setDraftModelKey}
                    modelSelectionHint={modelSelectionHint}
                    statusText={undefined}
                    statusError={undefined}
                  />
                </div>
              ) : null}
            </div>
          </div>

          {hasFiles ? (
            <aside
              data-testid="builder-files-rail"
              className={`flex min-h-0 shrink-0 bg-muted/10 ${filesExpanded ? "w-80" : "w-14"}`}
            >
              <div className="flex min-h-0 w-full flex-col">
                <div className={`p-2 ${filesExpanded ? "flex items-center justify-between" : "flex justify-center"}`}>
                  <Button
                    data-testid="builder-files-rail-toggle"
                    variant="ghost"
                    size="icon"
                    onClick={() => setFilesExpanded((current) => !current)}
                  >
                    {filesExpanded ? <ChevronRight className="h-4 w-4" /> : <FileCode2 className="h-4 w-4" />}
                    <span className="sr-only">Toggle files rail</span>
                  </Button>
                  {filesExpanded ? (
                    <Button variant="ghost" size="sm" onClick={handleDownloadFiles}>
                      Download
                    </Button>
                  ) : null}
                </div>

                {filesExpanded ? (
                  <div className="flex min-h-0 flex-1 flex-col">
                    <div data-testid="builder-files-list" className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
                      {currentPath !== "/" ? (
                        <button
                          type="button"
                          className="mb-1 flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                          onClick={() => {
                            const parentSegments = currentPath.split("/").filter(Boolean)
                            parentSegments.pop()
                            const nextPath = parentSegments.length > 0 ? `/${parentSegments.join("/")}/` : "/"
                            setCurrentPath(nextPath)
                            setSelectedFile(null)
                            setFileContent(null)
                          }}
                        >
                          <ChevronLeft className="h-4 w-4" />
                          <span>Parent directory</span>
                        </button>
                      ) : null}

                      {filesData?.files.map((file) => (
                        <button
                          key={`${currentPath}${file.name}`}
                          type="button"
                          className="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                          onClick={() => {
                            void handleSelectFile(file)
                          }}
                        >
                          {file.type === "dir" ? (
                            <Folder className="h-4 w-4 shrink-0" />
                          ) : (
                            <FileText className="h-4 w-4 shrink-0" />
                          )}
                          <span className="truncate">{file.name}</span>
                        </button>
                      ))}
                    </div>

                    {selectedFile ? (
                      <div className="min-h-0 px-3 py-3">
                        <div className="mb-2 flex items-center gap-2">
                          <FileText className="h-4 w-4 shrink-0" />
                          <span className="truncate text-sm font-medium">{selectedFile.name}</span>
                        </div>
                        {fileContent === null ? (
                          <div className="flex items-center text-sm text-muted-foreground">
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Loading preview…
                          </div>
                        ) : (
                          <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all text-xs text-muted-foreground">
                            {fileContent}
                          </pre>
                        )}
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </aside>
          ) : null}
        </section>
      </div>
    </div>
  )
}

function ConversationMessage({ message }: { message: BuilderMessage }) {
  if (message.role === "system") {
    return <div className="text-sm text-muted-foreground">{message.content}</div>
  }

  const isUserMessage = message.role === "user"

  return (
    <div className={`flex gap-2.5 ${isUserMessage ? "flex-row-reverse" : "flex-row"}`}>
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {isUserMessage ? <User className="h-3.5 w-3.5" /> : <Bot className="h-3.5 w-3.5" />}
      </div>
      <div
        className={`max-w-3xl rounded-[1.25rem] px-4 py-2.5 text-sm ${isUserMessage ? "bg-primary text-primary-foreground shadow-sm" : "bg-muted/55 text-foreground"
          }`}
      >
        {message.content}
      </div>
    </div>
  )
}

function StreamingConversationMessage({
  run,
  streamingLog,
}: {
  run: BuilderRun
  streamingLog: string
}) {
  const statusLabel = run.status === "queued" ? "Builder queued the next run" : "Builder is working"
  const hasLog = streamingLog.trim().length > 0

  return (
    <div className="flex gap-2.5">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Bot className="h-3.5 w-3.5" />
      </div>
      <div
        data-testid="builder-streaming-message"
        className="max-w-3xl rounded-[1.25rem] bg-muted/55 px-4 py-3 text-sm text-foreground"
      >
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>{statusLabel}</span>
        </div>
        <div className="mt-2">
          {hasLog ? (
            <pre className="whitespace-pre-wrap break-words text-sm text-foreground">{streamingLog}</pre>
          ) : (
            <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:120ms]" />
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:240ms]" />
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
