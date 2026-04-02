import { getStoredAccessToken, syncAuthCookie } from "@/lib/auth-session"
import * as React from "react"

interface LogViewerProps {
  appId: string
  instanceName: string
  containerName: string
}

export function LogViewer({ appId, instanceName, containerName }: LogViewerProps) {
  const [logs, setLogs] = React.useState<string[]>([])
  const logEndRef = React.useRef<HTMLDivElement>(null)
  React.useEffect(() => {
    const token = getStoredAccessToken()
    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    syncAuthCookie()
    const ws = new WebSocket(`${protocol}//${host}/api/v1/apps/${appId}/instances/${instanceName}/logs?container=${containerName}`)

    ws.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    return () => ws.close()
  }, [appId, instanceName, containerName])

  React.useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [logs])

  return (
    <div className="bg-black text-white p-4 font-mono text-sm h-100 overflow-y-auto rounded-lg">
      {logs.map((log, i) => (
        <div key={i}>{log}</div>
      ))}
      <div ref={logEndRef} />
    </div>
  )
}
