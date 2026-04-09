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
} from "@/api/builder-sessions"
import { envsApi, type Env } from "@/api/envs"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import {
  MessageSquare,
  Orbit,
  Sparkles,
  ChevronsUpDown,
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const AUTO_RESUME_WINDOW_MS = 8 * 60 * 60 * 1000
const HIDDEN_SYSTEM_MESSAGE_CONTENT = "run completed: replied without workspace changes"

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

export function useBuilderWorkspaceSession() {
  const { projectId: projectIdFromParams, sessionId } = useParams<{ projectId?: string; sessionId?: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const currentUserId = useAuthStore((state) => state.user?.id ?? "")
  const activeProjectId = useProjectStore((state) => state.activeProjectId)
  const activeProjectName = useProjectStore((state) => state.activeProjectName)
  const activeEnvName = useProjectStore((state) => state.activeEnvName)
  const setActiveContextWithNames = useProjectStore((state) => state.setActiveContextWithNames)
  const projectId = projectIdFromParams ?? activeProjectId ?? undefined

  const [messageInput, setMessageInput] = React.useState("")
  const [draftBuildEnvId, setDraftBuildEnvId] = React.useState("")
  const [draftModelKey, setDraftModelKey] = React.useState<string | null>(null)
  const [draftError, setDraftError] = React.useState("")
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
  const [previewLaunch, setPreviewLaunch] = React.useState<BuilderPreviewLaunch | null>(null)
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

    navigate(`/builder-sessions/${resumableSession.id}`, { replace: true })
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
  const selectedRuns = React.useMemo(
    () => selectedDetail?.runs ?? [],
    [selectedDetail?.runs]
  )
  const rawMessages = React.useMemo(
    () => selectedDetail?.messages ?? [],
    [selectedDetail?.messages]
  )

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
    setOptimisticQueuedMessages([])
    setStreamingRunId(null)
    setStreamingLog("")
    setPreviewLaunch(null)
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
      navigate(`/builder-sessions/${detail.session.id}`)
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
    setOptimisticQueuedMessages((current) => {
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
    })
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
  }, [activeStreamingRun, projectId, sessionId])

  const isSubmitting = createSessionMutation.isPending
  const isRedirectingToFreshSession = !sessionId && !isSessionsLoading && !!resumableSession && !isDraftOverride

  const handleComposerInputChange = React.useCallback((value: string) => {
    setMessageInput(value)
  }, [])

  const handleOpenDraft = React.useCallback(() => {
    if (!projectId) {
      return
    }

    navigate(`/builder-sessions?draft=1`)
  }, [navigate, projectId])

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
                    navigate(`/builder-sessions?draft=1`)
                    return
                  }

                  navigate(`/builder-sessions?draft=1`, { replace: !isDraftOverride })
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
      { label: "Builder", href: "/builder-sessions", icon: Sparkles },
      { label: environmentLabel, icon: Orbit, dropdown: environmentDropdown },
      { label: "Workspace", icon: MessageSquare },
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

  return {
    projectId,
    sessionId,
    navigate,
    sessions,
    isSessionsLoading,
    buildEnvs,
    draftBuildEnvOptions,
    draftBuildEnvId,
    setDraftBuildEnvId,
    draftModelOptions,
    draftModelKey,
    setDraftModelKey,
    modelSelectionHint,
    selectedDetail,
    selectedSession,
    selectedPreview,
    selectedRuns,
    latestRun,
    isSessionLoading,
    sessionError,
    visibleMessages,
    queuedMessages,
    activeStreamingRun,
    isAnyRunActive,
    hasFiles,
    sessionExports,
    exportPromotionPlan,
    promotingExportId,
    setPromotingExportId,
    promotionForm,
    setPromotionForm,
    buildPromotionForm,
    setBuildPromotionForm,
    deployBuildForm,
    setDeployBuildForm,
    streamingRunId,
    streamingLog,
    previewLaunch,
    messageInput,
    draftError,
    composerRef,
    isSubmitting,
    shouldShowConversationLoader,
    shouldShowSessionError,
    composerStatusText,
    selectedBuildEnvironment,
    breadcrumbItems,
    currentUserId,
    handleComposerInputChange,
    handleSendMessage,
    handleOpenDraft,
    handleDownloadPreview,
    handleOpenPreview,
    handleDownloadExport,
    handlePromoteExport,
    handlePromoteExportToBuild,
    handleDeployExportBuild,
    createExportMutation,
    promoteExportMutation,
    promoteExportToBuildMutation,
    deployExportBuildMutation,
    createSessionMutation,
    sendMessageMutation,
    getMessageKey,
  }
}
