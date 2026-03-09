import { codeRepositoriesApi } from "@/api/code-repositories"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { useTheme } from "@/components/theme-provider/theme-provider"
import Editor from "@monaco-editor/react"
import { useQuery } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"

interface BuildLogViewerProps {
  buildId: string
  repoId: string
}

export function BuildLogViewer({ buildId, repoId }: BuildLogViewerProps) {
  const [logs, setLogs] = React.useState<string[]>([])
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
    queryKey: ['build', buildId, repoId],
    queryFn: () => codeRepositoriesApi.getBuild(repoId, buildId),
    refetchInterval: 3000,
    enabled: !!repoId,
  })

  React.useEffect(() => {
    if (!buildId || !repoId) return
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
    const url = `${apiBase}/v1/code-repositories/${repoId}/builds/${buildId}/logs?token=${encodeURIComponent(token)}`

    const eventSource = new EventSource(url)

    eventSource.addEventListener('log', (event) => {
      setLogs((prev) => [...prev, event.data])
    })

    eventSource.addEventListener('done', () => {
      eventSource.close()
    })

    eventSource.onerror = () => {
      eventSource.close()
    }

    return () => {
      eventSource.close()
    }
  }, [buildId, repoId])

  return (
    <div className="space-y-4">
      {/* {build && ( */}
      <div className="flex items-center gap-4 text-xs">
        <span className="text-muted-foreground">Build #{build?.build_number}</span>
        <BuildStatusBadge status={build?.status || "unknown"} />
        {build?.git_ref && <span className="text-muted-foreground">Ref: {build?.git_ref}</span>}
        {build?.image_full_name && <span className="text-muted-foreground font-mono text-xs">{build?.image_full_name}</span>}
        {build?.error_message && <span className="text-destructive text-xs">{build?.error_message}</span>}
      </div>
      {/* )} */}
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
            <div className="flex items-center justify-center w-full h-full">
              <Loader2 className="h-6 w-6 animate-spin mr-2" />
              Loading editor...
            </div>
          }
        />
      </div>
    </div>
  )
}
