import { Plus, Terminal, X } from "lucide-react"
import * as React from "react"
import { Terminal as XTerm } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"
import "xterm/css/xterm.css"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface TerminalSession {
  id: string
  name: string
  isConnected: boolean
}

interface TerminalPanelProps {
  appId: string
  instanceName: string
  containerName: string
  targetType?: "pod" | "node"
}

export function TerminalPanel({ appId, instanceName, containerName, targetType = "pod" }: TerminalPanelProps) {
  const [sessions, setSessions] = React.useState<TerminalSession[]>([
    { id: "1", name: "Session 1", isConnected: false }
  ])
  const [activeSessionId, setActiveSessionId] = React.useState("1")

  const handleAddSession = () => {
    const newId = (sessions.length + 1).toString()
    setSessions([...sessions, { id: newId, name: `Session ${newId}`, isConnected: false }])
    setActiveSessionId(newId)
  }

  const handleRemoveSession = (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (sessions.length === 1) return
    const newSessions = sessions.filter(s => s.id !== id)
    setSessions(newSessions)
    if (activeSessionId === id) {
      setActiveSessionId(newSessions[0].id)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-2 py-1 border-b bg-muted/20">
        <div className="flex items-center gap-1 overflow-x-auto no-scrollbar max-w-[70%]">
          {sessions.map(session => (
            <div
              key={session.id}
              onClick={() => setActiveSessionId(session.id)}
              className={cn(
                "flex items-center gap-2 px-3 py-1.5 rounded-t-sm text-[10px] cursor-pointer transition-colors",
                activeSessionId === session.id
                  ? "bg-secondary"
                  : ""
              )}
            >
              <Terminal className="h-3 w-3" />
              <span className="truncate max-w-20">{session.name}</span>
              {sessions.length > 1 && (
                <X
                  className="h-3 w-3 ml-1 hover:text-red-400"
                  onClick={(e) => handleRemoveSession(session.id, e)}
                />
              )}
            </div>
          ))}
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={handleAddSession}
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        </div>

        <div className="flex items-center gap-2 pr-2">
          <div
            className={cn(
              "flex items-center gap-1.5 px-2 py-1 rounded text-[10px]",
              sessions.find(s => s.id === activeSessionId)?.isConnected
                ? "bg-green-500/10 text-green-400"
                : "bg-red-500/10 text-red-400"
            )}
          >
            <span
              className={cn(
                "w-1 h-1 rounded-full",
                sessions.find(s => s.id === activeSessionId)?.isConnected ? "bg-green-500 animate-pulse" : "bg-red-500"
              )}
            />
            {sessions.find(s => s.id === activeSessionId)?.isConnected ? "Connected" : "Disconnected"}
          </div>
        </div>
      </div>

      <div className="flex-1 relative overflow-hidden">
        {sessions.map(session => (
          <div
            key={session.id}
            className={cn(
              "absolute inset-0",
              activeSessionId === session.id ? "visible" : "invisible"
            )}
          >
            <TerminalInstance
              appId={appId}
              instanceName={instanceName}
              containerName={containerName}
              targetType={targetType}
              onConnectionChange={(connected) => {
                setSessions(prev => prev.map(s => s.id === session.id ? { ...s, isConnected: connected } : s))
              }}
            />
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between px-4 py-1.5 border-t text-[10px]">
        <div className="flex items-center gap-3">
          <span>{targetType === "node" ? `Node: ${instanceName}` : `Container: ${containerName}`}</span>
        </div>
        <div className="flex items-center gap-2 font-mono">
          <kbd className="px-1 py-0.5 rounded text-[9px]">Ctrl+C</kbd>
          <span>to interrupt</span>
        </div>
      </div>
    </div>
  )
}

function TerminalInstance({
  appId,
  instanceName,
  containerName,
  targetType,
  onConnectionChange
}: TerminalPanelProps & { onConnectionChange: (connected: boolean) => void }) {
  const terminalRef = React.useRef<HTMLDivElement>(null)
  const xtermRef = React.useRef<XTerm | null>(null)
  const wsRef = React.useRef<WebSocket | null>(null)

  React.useEffect(() => {
    if (!terminalRef.current) return

    const xterm = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'monospace',
      lineHeight: 1.2,
      // theme: {
      //   background: "#0a0a0a",
      //   foreground: "#e4e4e7",
      //   cursor: "#22c55e",
      //   cursorAccent: "#0a0a0a",
      //   selectionBackground: "#27272a",
      //   black: "#18181b",
      //   red: "#ef4444",
      //   green: "#22c55e",
      //   yellow: "#eab308",
      //   blue: "#3b82f6",
      //   magenta: "#a855f7",
      //   cyan: "#06b6d4",
      //   white: "#e4e4e7",
      //   brightBlack: "#52525b",
      //   brightRed: "#f87171",
      //   brightGreen: "#4ade80",
      //   brightYellow: "#facc15",
      //   brightBlue: "#60a5fa",
      //   brightMagenta: "#c084fc",
      //   brightCyan: "#22d3ee",
      //   brightWhite: "#fafafa",
      // },
      disableStdin: false,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()

    xterm.loadAddon(fitAddon)
    xterm.loadAddon(webLinksAddon)
    xterm.open(terminalRef.current)

    xtermRef.current = xterm

    // Defer initial fit to ensure container is fully rendered
    setTimeout(() => {
      fitAddon.fit()
    }, 0)

    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch (e) { }
    }

    if (!token) {
      xterm.writeln("\r\n\x1b[38;5;196m● Authentication required\x1b[0m")
      return
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsHost = import.meta.env.DEV ? "localhost:8080" : window.location.host
    const path = targetType === "node"
      ? `/api/v1/clusters/${appId}/nodes/${instanceName}/exec?token=${token}`
      : `/api/v1/apps/${appId}/instances/${instanceName}/exec?container=${containerName}&token=${token}`

    const ws = new WebSocket(`${protocol}//${wsHost}${path}`)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      onConnectionChange(true)
      console.log("WebSocket connected in terminal session")
    }

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        const text = new TextDecoder().decode(event.data)
        xterm.write(text)
      } else if (typeof event.data === 'string') {
        xterm.write(event.data)
      }
    }

    ws.onerror = () => {
      onConnectionChange(false)
      console.error("WebSocket error in terminal session")
    }

    ws.onclose = () => {
      onConnectionChange(false)
      console.log("WebSocket closed in terminal session")
    }

    xterm.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    const handleResize = () => {
      fitAddon.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: xterm.cols, rows: xterm.rows }))
      }
    }

    window.addEventListener("resize", handleResize)
    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: xterm.cols, rows: xterm.rows }))
      }
    })
    resizeObserver.observe(terminalRef.current)

    return () => {
      window.removeEventListener("resize", handleResize)
      resizeObserver.disconnect()
      ws.close()
      xterm.dispose()
    }
  }, [appId, instanceName, containerName])

  return <div ref={terminalRef} className="h-full w-full rounded overflow-hidden" />
}

