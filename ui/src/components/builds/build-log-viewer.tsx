import Editor from "@monaco-editor/react"
import { useQuery } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"

import { buildsApi } from "@/api/builds"
import { codeRepositoriesApi } from "@/api/code-repositories"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { useTheme } from "@/components/theme-provider/theme-provider"

interface BuildLogViewerProps {
  appId?: string
  buildId: string
  repoId?: string
}

export function BuildLogViewer({ appId, buildId, repoId }: BuildLogViewerProps) {
  const [logs, setLogs] = React.useState<string[]>([])
  const [streaming, setStreaming] = React.useState(false)
  const editorRef = React.useRef<any>(null)

  const handleEditorDidMount = (editor: any) => {
    editorRef.current = editor
  }

  const { theme } = useTheme()

  // Resolved theme for Monaco
  const [monacoTheme, setMonacoTheme] = React.useState<"vs" | "vs-dark">("vs")
  React.useEffect(() => {
    const resolve = () => {
      if (theme === "dark") return "vs-dark" as const
      if (theme === "light") return "vs" as const
      return document.documentElement.classList.contains("dark")
        ? ("vs-dark" as const)
        : ("vs" as const)
    }
    setMonacoTheme(resolve())
    if (theme !== "system") return
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => setMonacoTheme(resolve())
    media.addEventListener("change", handler)
    return () => media.removeEventListener("change", handler)
  }, [theme])

  React.useEffect(() => {
    if (editorRef.current && logs.length > 0) {
      const lineCount = editorRef.current.getModel()?.getLineCount() || 1
      editorRef.current.revealLine(lineCount)
    }
  }, [logs])

  const { data: build } = useQuery({
    queryKey: ['build', buildId, repoId ?? appId],
    queryFn: () =>
      repoId
        ? codeRepositoriesApi.getBuild(repoId, buildId)
        : buildsApi.get(appId!, buildId),
    refetchInterval: 3000,
    enabled: !!(repoId || appId),
  })

  React.useEffect(() => {
    if (!buildId || (!repoId && !appId)) return
    setStreaming(true)
    setLogs([])

    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch (_) { }
    }

    const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
    const url = repoId
      ? `${apiBase}/v1/code-repositories/${repoId}/builds/${buildId}/logs?token=${encodeURIComponent(token)}`
      : `${apiBase}/v1/apps/${appId}/builds/${buildId}/logs?token=${encodeURIComponent(token)}`

    const eventSource = new EventSource(url)

    eventSource.addEventListener('log', (event) => {
      setLogs((prev) => [...prev, event.data])
    })

    eventSource.addEventListener('done', () => {
      setStreaming(false)
      eventSource.close()
    })

    eventSource.onerror = () => {
      setStreaming(false)
      eventSource.close()
    }

    return () => {
      eventSource.close()
    }
  }, [appId, buildId, repoId])

  return (
    <div className="space-y-4">
      {build && (
        <div className="flex items-center gap-4 text-xs">
          <span className="text-muted-foreground">Build #{build.build_number}</span>
          <BuildStatusBadge status={build.status} />
          {build.git_ref && <span className="text-muted-foreground">Ref: {build.git_ref}</span>}
          {build.image_full_name && <span className="text-muted-foreground font-mono text-xs">{build.image_full_name}</span>}
          {build.error_message && <span className="text-destructive text-xs">{build.error_message}</span>}
        </div>
      )}
      <div className="border rounded-lg overflow-hidden h-[70vh]">
        <Editor
          height="100%"
          language="text"
          theme={monacoTheme}
          value={logs.join('\n')}
          onMount={handleEditorDidMount}
          options={{
            readOnly: true,
            fontSize: 12,
            lineNumbers: 'on',
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on',
            automaticLayout: true,
            padding: { top: 16, bottom: 16 },
          }}
          loading={
            <div className="flex items-center justify-center h-full bg-zinc-950 text-zinc-500">
              <Loader2 className="h-6 w-6 animate-spin mr-2" />
              Loading editor...
            </div>
          }
        />
      </div>
      {streaming && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground px-2">
          <Loader2 className="h-3 w-3 animate-spin" />
          Streaming logs...
        </div>
      )}
    </div>
  )
}
