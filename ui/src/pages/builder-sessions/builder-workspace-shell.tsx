import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import {
  Bot,
  ChevronLeft,
  ChevronRight,
  FileCode2,
  FileText,
  Folder,
  Loader2,
  MessageSquarePlus,
  Send,
  User,
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import {
  builderRunStatusLabels,
  builderSessionsApi,
  type BuilderMessage,
  type BuilderRunStatus,
  type BuilderSession,
  type BuilderSessionDetail,
  type BuilderWorkspaceFile,
} from "@/api/builder-sessions"
import { envsApi, type Env } from "@/api/envs"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { useAuthStore } from "@/stores/auth"

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

function getSessionLabel(session: BuilderSession | null): string {
  if (!session) {
    return "New conversation"
  }

  const primaryLabel = session.title?.trim() || session.summary?.trim()
  return primaryLabel || session.id.slice(0, 8)
}

function getMessageKey(message: BuilderMessage): string {
  return message.id || `${message.role}-${message.created_at}`
}

export function BuilderWorkspaceShell() {
  const { projectId, sessionId } = useParams<{ projectId: string; sessionId?: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const currentUserId = useAuthStore((state) => state.user?.id ?? "")

  const [messageInput, setMessageInput] = React.useState("")
  const [draftBuildEnvId, setDraftBuildEnvId] = React.useState("")
  const [draftError, setDraftError] = React.useState("")
  const [filesExpanded, setFilesExpanded] = React.useState(false)
  const [currentPath, setCurrentPath] = React.useState("/")
  const [selectedFile, setSelectedFile] = React.useState<BuilderWorkspaceFile | null>(null)
  const [fileContent, setFileContent] = React.useState<string | null>(null)
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

  const { data: draftBuildEnvs = [] } = useQuery({
    queryKey: ["builder-build-envs", projectId],
    queryFn: async () => {
      const response = await envsApi.list(projectId!, { page: 1, page_size: 100 })
      return (response.items ?? []).filter((env: Env) => env.is_build_env)
    },
    enabled: !!projectId && !sessionId && (isDraftOverride || !resumableSession),
  })

  const draftBuildEnvOptions = React.useMemo(
    () => draftBuildEnvs.map((env) => ({ label: env.name, value: env.id })),
    [draftBuildEnvs]
  )

  React.useEffect(() => {
    if (sessionId || draftBuildEnvId || draftBuildEnvOptions.length !== 1) {
      return
    }

    setDraftBuildEnvId(draftBuildEnvOptions[0].value)
  }, [draftBuildEnvId, draftBuildEnvOptions, sessionId])

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
  const selectedSessionRunActive = isRunActive(selectedSession?.latest_run_status)
  const hasFiles =
    !!selectedDetail &&
    (((selectedDetail.artifacts?.length ?? 0) > 0) || ((selectedDetail.session.artifact_count ?? 0) > 0))

  const { data: filesData } = useQuery({
    queryKey: ["builder-files", projectId, sessionId, currentPath],
    queryFn: () => builderSessionsApi.listFiles(projectId!, sessionId!, currentPath),
    enabled: !!projectId && !!sessionId && hasFiles,
  })

  React.useEffect(() => {
    setFilesExpanded(false)
    setCurrentPath("/")
    setSelectedFile(null)
    setFileContent(null)
  }, [sessionId])

  const createSessionMutation = useMutation({
    mutationFn: (payload: { buildEnvId: string; prompt: string }) =>
      builderSessionsApi.create(projectId!, {
        build_env_id: payload.buildEnvId,
        prompt: payload.prompt,
      }),
    onSuccess: (detail) => {
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

  const sendMessageMutation = useMutation({
    mutationFn: (payload: { selectedSessionId: string; content: string }) =>
      builderSessionsApi.postMessage(projectId!, payload.selectedSessionId, { content: payload.content }),
    onSuccess: () => {
      setMessageInput("")
      queryClient.invalidateQueries({ queryKey: ["builder-session", projectId, sessionId] })
      queryClient.invalidateQueries({ queryKey: ["builder-sessions", projectId] })
    },
    onError: (error: unknown) => {
      const message = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to send message", { description: message })
    },
  })

  const isSubmitting = createSessionMutation.isPending || sendMessageMutation.isPending
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
      sendMessageMutation.mutate({ selectedSessionId: sessionId, content })
      return
    }

    if (!draftBuildEnvId) {
      setDraftError("Build environment is required")
      return
    }

    createSessionMutation.mutate({ buildEnvId: draftBuildEnvId, prompt: content })
  }, [createSessionMutation, draftBuildEnvId, messageInput, sendMessageMutation, sessionId])

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

  const historyItems = sessions.map((session) => {
    const isSelected = session.id === sessionId

    return (
      <button
        key={session.id}
        type="button"
        className={`flex w-full flex-col items-start px-3 py-3 text-left transition-colors ${
          isSelected ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
        }`}
        onClick={() => navigate(`/projects/${projectId}/builder-sessions/${session.id}`)}
      >
        <span className="truncate text-sm font-medium">{getSessionLabel(session)}</span>
        <span className="truncate text-xs text-muted-foreground">{new Date(session.last_activity_at).toLocaleString()}</span>
      </button>
    )
  })

  const conversationTitle = sessionId ? getSessionLabel(selectedSession) : "New conversation"
  const conversationDescription = sessionId
    ? "Continue working in this Builder session. New prompts will be queued even while a run is active."
    : "Start a draft conversation. Ketches will create a Builder session when you send the first message."

  const shouldShowConversationLoader = (isSessionsLoading && sessions.length === 0) || isRedirectingToFreshSession
  const shouldShowSessionError = !!sessionId && !!sessionError
  const messages = selectedDetail?.messages ?? []

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        items={
          sessionId
            ? [
                { label: "Projects", href: "/projects" },
                { label: "Builder", href: `/projects/${projectId}/builder-sessions` },
                { label: conversationTitle },
              ]
            : [{ label: "Projects", href: "/projects" }, { label: "Builder" }]
        }
      />

      <div className="flex min-h-0 flex-1 overflow-hidden border bg-background">
        <aside className="flex min-h-0 w-72 shrink-0 flex-col border-r bg-muted/20">
          <div className="px-3 py-3">
            <button
              type="button"
              onClick={handleOpenDraft}
              className={`flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm transition-colors ${
                sessionId ? "text-muted-foreground hover:bg-muted/60 hover:text-foreground" : "bg-background text-foreground"
              }`}
            >
              <MessageSquarePlus className="h-4 w-4 shrink-0" />
              <span className="truncate font-medium">New conversation</span>
            </button>
          </div>
          <Separator />
          <div data-testid="builder-session-history" className="flex min-h-0 flex-1 flex-col overflow-y-auto">
            {historyItems}
          </div>
        </aside>

        <section className="flex min-h-0 min-w-0 flex-1">
          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="border-b px-4 py-4">
              <div className="flex items-center gap-2">
                <h1 className="text-base font-semibold">{conversationTitle}</h1>
                {selectedSessionRunActive ? (
                  <span className="text-xs text-muted-foreground">
                    {builderRunStatusLabels[selectedSession?.latest_run_status ?? "queued"]}
                  </span>
                ) : null}
              </div>
              <p className="mt-1 text-sm text-muted-foreground">{conversationDescription}</p>
            </div>

            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
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
                  <div className="space-y-4">
                    {messages.length === 0 ? (
                      <div className="flex min-h-40 flex-col items-center justify-center text-center text-sm text-muted-foreground">
                        <Bot className="mb-3 h-6 w-6" />
                        No messages yet. Send the next instruction to keep building.
                      </div>
                    ) : (
                      messages.map((message) => <ConversationMessage key={getMessageKey(message)} message={message} />)
                    )}
                  </div>
                ) : (
                  <div className="space-y-5">
                    <div className="flex min-h-40 flex-col justify-center">
                      <Bot className="mb-3 h-6 w-6 text-muted-foreground" />
                      <h2 className="text-sm font-medium">New conversation</h2>
                      <p className="mt-1 text-sm text-muted-foreground">
                        Describe what you want to build. The draft stays unsaved until your first send.
                      </p>
                    </div>

                    <div className="max-w-md">
                      <Field>
                        <FieldLabel htmlFor="builder-draft-build-env">Build environment</FieldLabel>
                        <FieldContent>
                          <Combobox
                            value={draftBuildEnvId}
                            onValueChange={(value) => {
                              setDraftBuildEnvId(value ?? "")
                              setDraftError("")
                            }}
                            itemToStringLabel={(value: string) =>
                              draftBuildEnvOptions.find((option) => option.value === value)?.label ?? value ?? ""
                            }
                          >
                            <ComboboxInput
                              id="builder-draft-build-env"
                              placeholder="Select a build environment"
                              className="w-full"
                            />
                            <ComboboxContent>
                              <ComboboxList>
                                {draftBuildEnvOptions.length === 0 ? (
                                  <ComboboxItem value="__no-build-env__" disabled>
                                    No build environments available
                                  </ComboboxItem>
                                ) : (
                                  draftBuildEnvOptions.map((option) => (
                                    <ComboboxItem key={option.value} value={option.value}>
                                      {option.label}
                                    </ComboboxItem>
                                  ))
                                )}
                              </ComboboxList>
                            </ComboboxContent>
                          </Combobox>
                        </FieldContent>
                        <FieldDescription>
                          The first message creates the Builder session in the selected environment.
                        </FieldDescription>
                        {draftError ? <FieldError>{draftError}</FieldError> : null}
                      </Field>
                    </div>
                  </div>
                )}
              </div>

              <div className="border-t px-4 py-4">
                <div className="flex gap-3">
                  <Textarea
                    ref={composerRef}
                    data-testid="builder-composer"
                    className="min-h-24 resize-none"
                    placeholder="Describe what you want to build or change..."
                    value={messageInput}
                    onChange={(event) => handleComposerInputChange(event.target.value)}
                    onInput={(event) => handleComposerInputChange(event.currentTarget.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault()
                        handleSendMessage()
                      }
                    }}
                    disabled={isSubmitting}
                  />
                  <Button
                    data-testid="builder-send-message"
                    className="shrink-0 self-end"
                    onClick={handleSendMessage}
                    disabled={isSubmitting}
                  >
                    <Send className="h-4 w-4" />
                    <span className="sr-only">Send message</span>
                  </Button>
                </div>
              </div>
            </div>
          </div>

          {hasFiles ? (
            <aside
              data-testid="builder-files-rail"
              className={`flex min-h-0 shrink-0 border-l ${filesExpanded ? "w-96" : "w-14"}`}
            >
              <div className="flex min-h-0 w-full flex-col">
                <div className={`border-b p-2 ${filesExpanded ? "flex items-center justify-between" : "flex justify-center"}`}>
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
                      <div className="min-h-0 border-t px-3 py-3">
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
    <div className={`flex gap-3 ${isUserMessage ? "flex-row-reverse" : "flex-row"}`}>
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {isUserMessage ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
      </div>
      <div
        className={`max-w-3xl rounded-lg px-4 py-3 text-sm ${
          isUserMessage ? "bg-primary text-primary-foreground" : "bg-muted text-foreground"
        }`}
      >
        {message.content}
      </div>
    </div>
  )
}
