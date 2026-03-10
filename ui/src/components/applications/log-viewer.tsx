import { useAuthStore } from "@/stores/auth"
import * as React from "react"

interface LogViewerProps {
  appId: string
  instanceName: string
  containerName: string
}

export function LogViewer({ appId, instanceName, containerName }: LogViewerProps) {
  const [logs, setLogs] = React.useState<string[]>([])
  const containerRef = React.useRef<HTMLDivElement>(null)
  const autoFollowRef = React.useRef(true)
  const token = useAuthStore((state) => state.accessToken)

  const updateAutoFollowState = React.useCallback(() => {
    const container = containerRef.current
    if (!container) return

    const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight
    autoFollowRef.current = distanceToBottom <= 24
  }, [])

  React.useEffect(() => {
    autoFollowRef.current = true
    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const ws = new WebSocket(`${protocol}//${host}/api/v1/apps/${appId}/instances/${instanceName}/logs?container=${containerName}&token=${token}`)

    ws.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    return () => ws.close()
  }, [appId, instanceName, containerName, token])

  React.useEffect(() => {
    const container = containerRef.current
    if (!container || !autoFollowRef.current) return

    container.scrollTop = container.scrollHeight
  }, [logs])

  return (
    <div
      ref={containerRef}
      onScroll={updateAutoFollowState}
      className="bg-black text-white p-4 font-mono text-sm h-100 overflow-y-auto rounded-lg"
    >
      {logs.map((log, i) => (
        <div key={i}>{log}</div>
      ))}
    </div>
  )
}
