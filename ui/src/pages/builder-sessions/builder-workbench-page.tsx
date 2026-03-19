import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import {
  Bot,
  ChevronRight,
  Download,
  File,
  FileText,
  Folder,
  Loader2,
  Send,
  User,
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"

import {
  builderSessionsApi,
  builderRunStatusColors,
  builderRunStatusLabels,
  type BuilderMessage,
  type BuilderRun,
  type BuilderWorkspaceFile,
  type BuilderRunStatus,
} from "@/api/builder-sessions"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

function RunStatusBadge({ status }: { status: string }) {
  const color = builderRunStatusColors[status as BuilderRunStatus] || "bg-gray-100 text-gray-800"
  const label = builderRunStatusLabels[status as BuilderRunStatus] || status
  return <Badge className={`${color} text-xs font-medium shrink-0`}>{label}</Badge>
}

export function BuilderWorkbenchPage() {
  const { projectId, sessionId } = useParams<{ projectId: string; sessionId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  // --- Data fetching ---
  const { data: sessionDetail, isLoading, error } = useQuery({
    queryKey: ["builder-session", projectId, sessionId],
    queryFn: () => builderSessionsApi.get(projectId!, sessionId!),
    enabled: !!projectId && !!sessionId,
  })

  // Poll when a run is active
  const isRunActive =
    sessionDetail?.session.latest_run_status === "executing" ||
    sessionDetail?.session.latest_run_status === "queued"

  const { data: sessionDetailRefetch } = useQuery({
    queryKey: ["builder-session", projectId, sessionId],
    queryFn: () => builderSessionsApi.get(projectId!, sessionId!),
    enabled: isRunActive,
    refetchInterval: 3000,
  })

  // Use whichever has the latest data
  const activeDetail = sessionDetailRefetch ?? sessionDetail

  // --- Chat state ---
  const [messageInput, setMessageInput] = React.useState("")
  const chatScrollRef = React.useRef<HTMLDivElement>(null)

  React.useEffect(() => {
    if (chatScrollRef.current) {
      chatScrollRef.current.scrollTop = chatScrollRef.current.scrollHeight
    }
  }, [activeDetail?.messages])

  // --- File explorer state ---
  const [currentPath, setCurrentPath] = React.useState("/")
  const [selectedFile, setSelectedFile] = React.useState<BuilderWorkspaceFile | null>(null)
  const [fileContent, setFileContent] = React.useState<string | null>(null)

  const { data: filesData } = useQuery({
    queryKey: ["builder-files", projectId, sessionId, currentPath],
    queryFn: () => builderSessionsApi.listFiles(projectId!, sessionId!, currentPath),
    enabled: !!projectId && !!sessionId && sessionDetail?.session.status === "ready",
  })

  const loadFileContent = React.useCallback(
    async (file: BuilderWorkspaceFile) => {
      if (file.type === "dir") {
        setSelectedFile(null)
        setFileContent(null)
        setCurrentPath(file.name === "/" ? "/" : `${currentPath.replace(/\/$/, "")}/${file.name}/`)
        return
      }
      setSelectedFile(file)
      setFileContent(null)
      try {
        const resp = await builderSessionsApi.readFile(projectId!, sessionId!, `${currentPath}${file.name}`)
        setFileContent(resp.content)
      } catch {
        setFileContent("[Failed to load file content]")
      }
    },
    [projectId, sessionId, currentPath]
  )

  // --- Run log state ---
  const [selectedRunId, setSelectedRunId] = React.useState<string | null>(null)
  const [logLines, setLogLines] = React.useState<string[]>([])
  const logScrollRef = React.useRef<HTMLDivElement>(null)
  const eventSourceRef = React.useRef<EventSource | null>(null)

  React.useEffect(() => {
    if (logScrollRef.current) {
      logScrollRef.current.scrollTop = logScrollRef.current.scrollHeight
    }
  }, [logLines])

  const startLogStream = React.useCallback(
    (runId: string) => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
      setLogLines([])
      setSelectedRunId(runId)

      const baseUrl = import.meta.env.VITE_API_BASE_URL || "/api"
      const authData = localStorage.getItem("auth-storage")
      let token = ""
      if (authData) {
        try {
          const { state } = JSON.parse(authData)
          token = state.accessToken || ""
        } catch { /* ignore */ }
      }

      const url = `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/runs/${runId}/logs`
      const es = new EventSource(`${url}?token=${encodeURIComponent(token)}`)
      eventSourceRef.current = es

      es.addEventListener("log", (e) => {
        try {
          const data = JSON.parse(e.data)
          setLogLines((prev) => [...prev, data.line || e.data])
        } catch {
          setLogLines((prev) => [...prev, e.data])
        }
      })

      es.addEventListener("done", () => {
        es.close()
        eventSourceRef.current = null
        setLogLines((prev) => [...prev, "[stream closed]"])
        queryClient.invalidateQueries({ queryKey: ["builder-session", projectId, sessionId] })
      })

      es.onerror = () => {
        es.close()
        eventSourceRef.current = null
        setLogLines((prev) => [...prev, "[connection closed]"])
      }
    },
    [projectId, sessionId, queryClient]
  )

  React.useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [])

  // When a new run appears in the detail, stream its logs if selected
  React.useEffect(() => {
    const latestRun = activeDetail?.runs?.[activeDetail.runs.length - 1]
    if (latestRun && latestRun.status === "executing" && selectedRunId !== latestRun.id) {
      startLogStream(latestRun.id)
    }
  }, [activeDetail?.runs])

  // --- Mutations ---
  const sendMessageMutation = useMutation({
    mutationFn: () =>
      builderSessionsApi.postMessage(projectId!, sessionId!, { content: messageInput }),
    onSuccess: () => {
      setMessageInput("")
      queryClient.invalidateQueries({ queryKey: ["builder-session", projectId, sessionId] })
    },
    onError: (error: unknown) => {
      const msg = isAxiosError(error) ? error.response?.data?.error : "Unknown error"
      toast.error("Failed to send message", { description: msg })
    },
  })

  const handleSendMessage = () => {
    if (!messageInput.trim()) return
    sendMessageMutation.mutate()
  }

  const handleDownload = async () => {
    try {
      await builderSessionsApi.downloadTarBlob(projectId!, sessionId!)
    } catch (err) {
      toast.error("Download failed")
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center min-h-100">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !activeDetail) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center min-h-100 gap-4">
        <p className="text-muted-foreground">Failed to load session</p>
        <Button variant="outline" onClick={() => navigate("/projects")}>
          Back to projects
        </Button>
      </div>
    )
  }

  const { session, messages, runs } = activeDetail
  const latestRun = runs[runs.length - 1]

  return (
    <div className="flex flex-col flex-1 gap-4 min-h-0">
      <PageHeader
        items={[
          { label: "Projects", href: "/projects" },
          { label: "Builder Sessions", href: `/projects/${projectId}/builder-sessions` },
          { label: session.title || session.id.slice(0, 8) },
        ]}
      />

      {/* Three-panel workbench */}
      <div className="flex flex-1 gap-4 min-h-0 overflow-hidden">
        {/* Left: Chat */}
        <div className="flex flex-col w-1/3 min-w-0 border rounded-lg bg-card overflow-hidden">
          <div className="px-4 py-3 border-b shrink-0">
            <h3 className="text-sm font-semibold">Conversation</h3>
          </div>
          <div className="flex-1 overflow-y-auto" ref={chatScrollRef as React.RefObject<HTMLDivElement>}>
            <div className="flex flex-col gap-3 p-4">
              {messages.length === 0 && (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <Bot className="h-8 w-8 text-muted-foreground mb-2" />
                  <p className="text-sm text-muted-foreground">
                    Send a message to start building your app.
                  </p>
                </div>
              )}
              {messages.map((msg: BuilderMessage) => (
                <MessageBubble key={msg.id} message={msg} />
              ))}
              {isRunActive && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground pl-3">
                  <Loader2 className="h-3 w-3 animate-spin shrink-0" />
                  <span>AI is generating...</span>
                </div>
              )}
            </div>
          </div>
          <div className="border-t p-3 shrink-0 space-y-2">
            {latestRun && latestRun.status !== "completed" && latestRun.status !== "succeeded" && (
              <div className="flex items-center gap-2">
                <RunStatusBadge status={latestRun.status} />
                {latestRun.error_message && (
                  <span className="text-xs text-destructive truncate">{latestRun.error_message}</span>
                )}
              </div>
            )}
            <div className="flex gap-2">
              <Textarea
                className="min-h-10 max-h-32 resize-none"
                placeholder="Describe what you want to add or change..."
                value={messageInput}
                onChange={(e) => setMessageInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault()
                    handleSendMessage()
                  }
                }}
                disabled={sendMessageMutation.isPending || isRunActive}
              />
              <Button
                size="icon"
                className="shrink-0"
                onClick={handleSendMessage}
                disabled={!messageInput.trim() || sendMessageMutation.isPending || isRunActive}
              >
                <Send className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        {/* Middle: File Explorer */}
        <div className="flex flex-col w-1/3 min-w-0 border rounded-lg bg-card overflow-hidden">
          <div className="px-4 py-3 border-b shrink-0 flex items-center justify-between">
            <h3 className="text-sm font-semibold">Files</h3>
            <Button variant="ghost" size="icon-sm" onClick={handleDownload} title="Download all">
              <Download className="h-3.5 w-3.5" />
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto">
            <div className="p-2">
              {/* Breadcrumb */}
              {currentPath !== "/" && (
                <button
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded hover:bg-muted w-full text-left"
                  onClick={() => {
                    const parts = currentPath.split("/").filter(Boolean)
                    parts.pop()
                    const parent = "/" + parts.join("/") + (parts.length ? "/" : "")
                    setCurrentPath(parent)
                    setSelectedFile(null)
                    setFileContent(null)
                  }}
                >
                  <ChevronRight className="h-3 w-3 rotate-180" />
                  <span>..</span>
                </button>
              )}
              {filesData?.files.map((file) => (
                <button
                  key={file.name}
                  className="flex items-center gap-2 text-sm hover:bg-muted px-2 py-1.5 rounded w-full text-left"
                  onClick={() => loadFileContent(file)}
                >
                  {file.type === "dir" ? (
                    <Folder className="h-4 w-4 text-yellow-500 shrink-0" />
                  ) : (
                    <File className="h-4 w-4 text-muted-foreground shrink-0" />
                  )}
                  <span className="truncate">{file.name}</span>
                </button>
              ))}
              {filesData?.files.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-4">
                  No files yet. Run the AI to generate files.
                </p>
              )}
            </div>
          </div>

          {/* File preview */}
          {selectedFile && (
            <div className="border-t flex flex-col shrink-0 max-h-64">
              <div className="px-4 py-2 border-b flex items-center gap-2">
                <FileText className="h-3.5 w-3.5 shrink-0" />
                <span className="text-xs font-medium truncate">{selectedFile.name}</span>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="ml-auto shrink-0"
                  onClick={() => {
                    setSelectedFile(null)
                    setFileContent(null)
                  }}
                >
                  ×
                </Button>
              </div>
              <div className="flex-1 overflow-y-auto">
                {fileContent === null ? (
                  <div className="flex items-center justify-center py-4">
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  </div>
                ) : (
                  <pre className="text-xs p-3 whitespace-pre-wrap break-all font-mono text-muted-foreground max-h-48 overflow-auto">
                    {fileContent}
                  </pre>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Right: Runs */}
        <div className="flex flex-col w-1/3 min-w-0 border rounded-lg bg-card overflow-hidden">
          <div className="px-4 py-3 border-b shrink-0">
            <h3 className="text-sm font-semibold">Runs</h3>
          </div>
          <div className="flex-1 overflow-y-auto">
            <div className="flex flex-col divide-y">
              {runs.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-6">
                  No runs yet.
                </p>
              )}
              {runs.map((run: BuilderRun) => (
                <div key={run.id} className="p-3 space-y-1.5">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-mono text-muted-foreground truncate">
                      {run.id.slice(0, 8)}
                    </span>
                    <RunStatusBadge status={run.status} />
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>{formatDate(run.created_at)}</span>
                    {run.started_at && (
                      <>
                        <span>→</span>
                        <span>{run.completed_at ? formatDate(run.completed_at) : "running"}</span>
                      </>
                    )}
                  </div>
                  {run.instruction_summary && (
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {run.instruction_summary}
                    </p>
                  )}
                  {run.error_message && (
                    <p className="text-xs text-destructive">{run.error_message}</p>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full h-7 text-xs"
                    onClick={() => startLogStream(run.id)}
                  >
                    {selectedRunId === run.id ? "Hide logs" : "View logs"}
                  </Button>

                  {/* Inline log viewer */}
                  {selectedRunId === run.id && (
                    <div className="mt-1 border rounded bg-black/5 p-2 max-h-48 overflow-auto">
                      {logLines.length === 0 ? (
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <Loader2 className="h-3 w-3 animate-spin" />
                          <span>Connecting...</span>
                        </div>
                      ) : (
                        logLines.map((line, i) => (
                          <LogLine key={i} line={line} />
                        ))
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function MessageBubble({ message }: { message: BuilderMessage }) {
  if (message.role === "system") {
    return (
      <div className="bg-muted/50 rounded-lg px-3 py-2 text-xs text-muted-foreground italic">
        {message.content}
      </div>
    )
  }

  const isUser = message.role === "user"
  return (
    <div className={`flex gap-2 ${isUser ? "flex-row-reverse" : "flex-row"}`}>
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted">
        {isUser ? <User className="h-3.5 w-3.5" /> : <Bot className="h-3.5 w-3.5" />}
      </div>
      <div
        className={`max-w-[80%] rounded-lg px-3 py-2 text-sm ${
          isUser ? "bg-primary text-primary-foreground" : "bg-muted"
        }`}
      >
        {message.content}
      </div>
    </div>
  )
}

function LogLine({ line }: { line: string }) {
  const prefix = line.startsWith("[system]") || line.startsWith("[agent]") ? line.split("]")[0] + "]" : null
  const rest = prefix ? line.slice(prefix.length) : line

  let colorClass = "text-muted-foreground"
  if (line.startsWith("[system]")) colorClass = "text-blue-500"
  else if (line.startsWith("[agent]")) colorClass = "text-green-600"
  else if (line.startsWith("[error]")) colorClass = "text-red-500"

  return (
    <div className="text-xs font-mono whitespace-pre-wrap break-all">
      {prefix && (
        <span className={colorClass}>
          {prefix}
          <span className="text-muted-foreground">{" "}</span>
        </span>
      )}
      <span className="text-muted-foreground">{rest}</span>
    </div>
  )
}
