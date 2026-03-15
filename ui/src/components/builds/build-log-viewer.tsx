import { codeRepositoriesApi } from "@/api/code-repositories"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { parseBuildLogAnsi, type BuildLogDecoration } from "@/lib/build-log-ansi"
import Editor, { type Monaco, type OnMount } from "@monaco-editor/react"
import { useQuery } from "@tanstack/react-query"
import { AlertTriangle, Frame, GitBranch, Image, Loader2 } from "lucide-react"
import type { editor } from "monaco-editor"
import * as React from "react"

interface BuildLogViewerProps {
  buildId: string
  repoId: string
}

export function BuildLogViewer({ buildId, repoId }: BuildLogViewerProps) {
  const [logs, setLogs] = React.useState<string[]>([])
  const [editorReady, setEditorReady] = React.useState(false)
  const editorRef = React.useRef<editor.IStandaloneCodeEditor | null>(null)
  const monacoRef = React.useRef<Monaco | null>(null)
  const containerRef = React.useRef<HTMLDivElement>(null)
  const autoFollowRef = React.useRef(true)
  const hasInitializedTailRef = React.useRef(false)
  const decorationIdsRef = React.useRef<string[]>([])
  const parsedLog = React.useMemo(() => parseBuildLogAnsi(logs.join("")), [logs])

  const scheduleScrollToBottom = React.useCallback((force = false) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (!editorRef.current) return
        if (!force && !autoFollowRef.current) return
        editorRef.current.setScrollTop(editorRef.current.getScrollHeight())
        if (force) {
          hasInitializedTailRef.current = true
          autoFollowRef.current = true
        }
      })
    })
  }, [])

  const isNearBottom = React.useCallback((editor: editor.IStandaloneCodeEditor) => {
    const viewportHeight = editor.getLayoutInfo()?.height ?? 0
    const scrollTop = editor.getScrollTop?.() ?? 0
    const scrollHeight = editor.getScrollHeight?.() ?? 0
    // Keep following when user is within a small threshold from the tail.
    return scrollTop + viewportHeight >= scrollHeight - 24
  }, [])

  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor
    monacoRef.current = monaco
    setEditorReady(true)
    autoFollowRef.current = true
    hasInitializedTailRef.current = false
    editor.onDidScrollChange((e: { scrollTopChanged: boolean }) => {
      // Ignore non-user-like scroll changes (e.g. content height changes on new logs).
      if (!e?.scrollTopChanged) return
      if (!hasInitializedTailRef.current) return
      autoFollowRef.current = isNearBottom(editor)
    })

    editor.onDidLayoutChange?.(() => {
      if (!autoFollowRef.current) return
      scheduleScrollToBottom()
    })

    editor.onDidContentSizeChange?.(() => {
      if (!autoFollowRef.current) return
      scheduleScrollToBottom()
    })
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
    if (!editorReady || parsedLog.text.length === 0) return

    if (!hasInitializedTailRef.current) {
      scheduleScrollToBottom(true)
      return
    }

    if (autoFollowRef.current) {
      scheduleScrollToBottom()
    }
  }, [parsedLog.text, editorReady, scheduleScrollToBottom])

  React.useEffect(() => {
    const editor = editorRef.current
    const monaco = monacoRef.current
    if (!editor || !monaco || !editor.getModel()) {
      return
    }

    decorationIdsRef.current = applyAnsiDecorations(
      editor,
      monaco,
      decorationIdsRef.current,
      parsedLog.decorations,
    )
  }, [parsedLog, editorReady])

  React.useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const observer = new ResizeObserver(() => {
      if (!autoFollowRef.current || parsedLog.text.length === 0) return
      scheduleScrollToBottom()
    })
    observer.observe(container)

    return () => observer.disconnect()
  }, [parsedLog.text.length, scheduleScrollToBottom])

  const { data: build } = useQuery({
    queryKey: ['build', buildId, repoId],
    queryFn: () => codeRepositoriesApi.getBuild(repoId, buildId),
    refetchInterval: 3000,
    enabled: !!repoId,
  })

  React.useEffect(() => {
    if (!buildId || !repoId) return
    setLogs([])
    autoFollowRef.current = true
    hasInitializedTailRef.current = false
    decorationIdsRef.current = []

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
    <div className="space-y-2">
      {/* {build && ( */}
      <div className="flex items-center gap-4 text-xs">
        <span className="flex items-center text-muted-foreground">Build<Frame className="ml-4 mr-1 h-3 w-3 inline" />{build?.build_number}</span>
        <BuildStatusBadge status={build?.status || "unknown"} />
        {build?.git_ref && <span className="flex items-center text-muted-foreground"> <GitBranch className="mr-1 h-3 w-3 inline" /><span>{build?.git_ref}</span></span>}
        {build?.image_full_name && <span className="flex items-center text-muted-foreground font-mono text-xs"><Image className="mr-1 h-3 w-3 inline" /><span>{build?.image_full_name}</span></span>}
        {build?.error_message && <span className="flex items-center text-destructive text-xs"><AlertTriangle className="mr-1 h-3 w-3 inline" /><span>{build?.error_message}</span></span>}
      </div>
      {/* )} */}
      <div ref={containerRef} className="border rounded-lg overflow-hidden h-[70vh]">
        <Editor
          height="100%"
          language="text"
          theme={monacoTheme}
          value={parsedLog.text}
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

function applyAnsiDecorations(
  editor: editor.IStandaloneCodeEditor,
  monaco: Monaco,
  previousDecorationIds: string[],
  decorations: BuildLogDecoration[],
): string[] {
  return editor.deltaDecorations(
    previousDecorationIds,
    decorations.map((decoration) => ({
      range: new monaco.Range(
        decoration.startLineNumber,
        decoration.startColumn,
        decoration.endLineNumber,
        decoration.endColumn,
      ),
      options: {
        inlineClassName: decoration.inlineClassName,
        stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
      },
    })),
  )
}
