import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Separator } from "@/components/ui/separator"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { Plus, Terminal, X } from "lucide-react"
import * as React from "react"
import { Terminal as XTerm } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"
import "xterm/css/xterm.css"
import { WorkloadPanelFrame } from "./workload-panel-frame"

interface TerminalSession {
  id: string
  name: string
  connectionState: "connecting" | "connected" | "disconnected"
  reconnectNonce: number
}

interface TerminalPanelProps {
  appId: string
  instanceName: string
  containerName: string
  targetType?: "pod" | "node"
}

export function TerminalPanel({ appId, instanceName, containerName, targetType = "pod" }: TerminalPanelProps) {
  const [sessions, setSessions] = React.useState<TerminalSession[]>([
    { id: "1", name: "Session 1", connectionState: "connecting", reconnectNonce: 0 }
  ])
  const [activeSessionId, setActiveSessionId] = React.useState("1")

  const handleAddSession = () => {
    const newId = (sessions.length + 1).toString()
    setSessions([...sessions, { id: newId, name: `Session ${newId}`, connectionState: "connecting", reconnectNonce: 0 }])
    setActiveSessionId(newId)
  }

  const handleReconnectSession = React.useCallback((id: string) => {
    setSessions(prev => prev.map(session => (
      session.id === id
        ? { ...session, connectionState: "connecting", reconnectNonce: session.reconnectNonce + 1 }
        : session
    )))
  }, [])

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
    <WorkloadPanelFrame
      toolbar={(
        <>
          <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto no-scrollbar">
            {sessions.map(session => (
              <React.Fragment key={session.id}>
                <div
                  onClick={() => setActiveSessionId(session.id)}
                  className={cn(
                    "flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-[10px] transition-colors",
                    activeSessionId === session.id ? "bg-secondary" : "",
                  )}
                >
                  <span
                    className={cn(
                      "h-1.5 w-1.5 rounded-full",
                      session.connectionState === "connected" && "bg-green-500",
                      session.connectionState === "connecting" && "bg-amber-500 animate-pulse",
                      session.connectionState === "disconnected" && "bg-red-500"
                    )}
                    title={session.connectionState}
                  />
                  <Terminal className="h-3 w-3" />
                  <span className="max-w-20 truncate">{session.name}</span>
                  {sessions.length > 1 && (
                    <X
                      className="ml-1 h-3 w-3 hover:text-red-500"
                      onClick={(e) => handleRemoveSession(session.id, e)}
                    />
                  )}
                </div>
                <Separator orientation="vertical" className="mt-1 mb-1 mx-1" />
              </React.Fragment>
            ))}
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={handleAddSession}
                  />
                }
              >
                <Plus className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>New terminal session</TooltipContent>
            </Tooltip>
          </div>
        </>
      )}
      status={(
        <>
          <div>{targetType === "node" ? `Node: ${instanceName}` : `Container: ${containerName}`}</div>
          <div className="flex items-center gap-2 font-mono">
            <kbd className="rounded px-1 py-0.5 text-[9px]">Ctrl+C</kbd>
            <span>to interrupt</span>
          </div>
        </>
      )}
    >
      <div className="relative h-full min-h-0 overflow-hidden bg-zinc-950">
        {sessions.map(session => (
          <div
            key={session.id}
            className={cn(
              "absolute inset-0 min-h-0",
              activeSessionId === session.id ? "visible" : "invisible"
            )}
          >
            <div className="h-full min-h-0 pl-1">
              <div className="relative h-full min-h-0 rounded-md shadow-inner">
                <TerminalInstance
                  key={`${session.id}-${session.reconnectNonce}`}
                  appId={appId}
                  instanceName={instanceName}
                  containerName={containerName}
                  targetType={targetType}
                  onConnectionStateChange={(state) => {
                    setSessions(prev => prev.map(s => s.id === session.id ? { ...s, connectionState: state } : s))
                  }}
                />
                {activeSessionId === session.id && session.connectionState === "disconnected" && (
                  <div className="absolute inset-0 z-10 p-4 bg-zinc-950">
                    {/* <EmptyState
                      title="Terminal disconnected"
                      description="The terminal connection was closed. Reconnect to open a new shell session."
                      // icon={Terminal}
                      actionText="Reconnect"
                      actionIcon={RefreshCw}
                      onAction={() => handleReconnectSession(session.id)}
                      border={false}
                      className="h-full"
                    /> */}
                    <Empty className="h-full border border-dashed border-amber-500/50 bg-amber-500/10">
                      <EmptyHeader>
                        <EmptyMedia variant="icon" className="bg-gray-500/20">
                          <Terminal className="text-gray-500 " />
                        </EmptyMedia>
                        <EmptyTitle className="text-amber-500">Terminal disconnected</EmptyTitle>
                        <EmptyDescription>The terminal connection was closed. Reconnect to open a new shell session.</EmptyDescription>
                      </EmptyHeader>
                      <EmptyContent>
                        <Button onClick={() => handleReconnectSession(session.id)}>Reconnect</Button>
                      </EmptyContent>
                    </Empty>
                  </div>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </WorkloadPanelFrame>
  )
}

function TerminalInstance({
  appId,
  instanceName,
  containerName,
  targetType,
  onConnectionStateChange
}: TerminalPanelProps & { onConnectionStateChange: (state: "connecting" | "connected" | "disconnected") => void }) {
  const terminalRef = React.useRef<HTMLDivElement>(null)
  const xtermRef = React.useRef<XTerm | null>(null)
  const wsRef = React.useRef<WebSocket | null>(null)
  const onConnectionStateChangeRef = React.useRef(onConnectionStateChange)

  React.useEffect(() => {
    onConnectionStateChangeRef.current = onConnectionStateChange
  }, [onConnectionStateChange])

  React.useEffect(() => {
    if (!terminalRef.current) return

    onConnectionStateChangeRef.current("connecting")


    let isDisposed = false
    let isIntentionalSocketClose = false
    let isTerminalOpened = false
    let initialFitTimer: ReturnType<typeof window.setTimeout> | null = null
    let connectTimer: ReturnType<typeof window.setTimeout> | null = null
    let openRetryTimer: ReturnType<typeof window.setTimeout> | null = null
    let openRaf: number | null = null

    const xterm = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'monospace',
      lineHeight: 1,
      theme: {
        background: "#09090b", // bg-zinc-950
        // foreground: "#e4e4e7",
        // cursor: "#22c55e",
        // cursorAccent: "#0a0a0a",
        // selectionBackground: "#27272a",
        // black: "#18181b",
        // red: "#ef4444",
        // green: "#22c55e",
        // yellow: "#eab308",
        // blue: "#3b82f6",
        // magenta: "#a855f7",
        // cyan: "#06b6d4",
        // white: "#e4e4e7",
        // brightBlack: "#52525b",
        // brightRed: "#f87171",
        // brightGreen: "#4ade80",
        // brightYellow: "#facc15",
        // brightBlue: "#60a5fa",
        // brightMagenta: "#c084fc",
        // brightCyan: "#22d3ee",
        // brightWhite: "#fafafa",
      },
      disableStdin: false,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()

    xterm.loadAddon(fitAddon)
    xterm.loadAddon(webLinksAddon)
    fitAddon.fit()

    const safeFit = () => {
      if (isDisposed || !isTerminalOpened) return
      try {
        fitAddon.fit()
      } catch {
        // xterm can throw during teardown races in development strict mode.
      }
    }

    const tryOpenTerminal = () => {
      if (isDisposed || isTerminalOpened) return

      const mountNode = terminalRef.current
      if (!mountNode || !mountNode.isConnected) return

      const rect = mountNode.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return

      try {
        xterm.open(mountNode)
        isTerminalOpened = true
        xtermRef.current = xterm
      } catch {
        // Retry open on the next frame if DOM/layout isn't stable yet.
      }
    }

    const scheduleOpenTerminal = () => {
      if (isDisposed || isTerminalOpened) return
      openRaf = window.requestAnimationFrame(() => {
        openRaf = null
        tryOpenTerminal()
        if (!isTerminalOpened && !isDisposed) {
          openRetryTimer = window.setTimeout(scheduleOpenTerminal, 50)
          return
        }

        safeFit()
        xterm.focus()
      })
    }

    scheduleOpenTerminal()

    // Defer initial fit to ensure container is fully rendered
    initialFitTimer = window.setTimeout(() => {
      safeFit()
    }, 0)

    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch { }
    }

    if (!token) {
      xterm.writeln("\r\n\x1b[38;5;196m● Authentication required\x1b[0m")
      onConnectionStateChangeRef.current("disconnected")
      return () => {
        isDisposed = true
        if (initialFitTimer !== null) {
          window.clearTimeout(initialFitTimer)
        }
        xterm.dispose()
        xtermRef.current = null
        wsRef.current = null
      }
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsHost = import.meta.env.DEV ? "localhost:8080" : window.location.host
    const path = targetType === "node"
      ? `/api/v1/clusters/${encodeURIComponent(appId)}/nodes/${encodeURIComponent(instanceName)}/exec?${new URLSearchParams({ token }).toString()}`
      : `/api/v1/apps/${encodeURIComponent(appId)}/instances/${encodeURIComponent(instanceName)}/exec?${new URLSearchParams({ container: containerName, token }).toString()}`
    const wsUrl = `${protocol}//${wsHost}${path}`

    const dataDisposable = xterm.onData((data) => {
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    const handleResize = () => {
      safeFit()
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: xterm.cols, rows: xterm.rows }))
      }
    }

    window.addEventListener("resize", handleResize)
    const resizeObserver = new ResizeObserver(() => {
      safeFit()
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: xterm.cols, rows: xterm.rows }))
      }
    })
    resizeObserver.observe(terminalRef.current)

    // Delay connect slightly so StrictMode's test mount/unmount cycle can be cancelled cleanly.
    connectTimer = window.setTimeout(() => {
      if (isDisposed) return

      const ws = new WebSocket(wsUrl)
      ws.binaryType = "arraybuffer"
      wsRef.current = ws

      ws.onopen = () => {
        if (isDisposed) {
          ws.close(1000, "Terminal instance disposed before open")
          return
        }
        safeFit()
        if (xterm.cols > 0 && xterm.rows > 0) {
          ws.send(JSON.stringify({ type: "resize", cols: xterm.cols, rows: xterm.rows }))
        }
        xterm.focus()
        onConnectionStateChangeRef.current("connected")
        console.log("WebSocket connected in terminal session")
      }

      ws.onmessage = (event) => {
        if (isDisposed) return
        if (event.data instanceof ArrayBuffer) {
          const text = new TextDecoder().decode(event.data)
          xterm.write(text)
        } else if (typeof event.data === "string") {
          xterm.write(event.data)
        }
      }

      ws.onerror = () => {
        if (isDisposed || isIntentionalSocketClose) {
          return
        }
        onConnectionStateChangeRef.current("disconnected")
        console.error("WebSocket error in terminal session")
      }

      ws.onclose = (event) => {
        onConnectionStateChangeRef.current("disconnected")
        if (!isDisposed && !isIntentionalSocketClose) {
          console.warn("WebSocket closed in terminal session", {
            code: event.code,
            reason: event.reason,
            wasClean: event.wasClean,
          })
        }
      }
    }, 120)

    return () => {
      isDisposed = true
      window.removeEventListener("resize", handleResize)
      resizeObserver.disconnect()
      dataDisposable.dispose()
      if (initialFitTimer !== null) {
        window.clearTimeout(initialFitTimer)
      }
      if (connectTimer !== null) {
        window.clearTimeout(connectTimer)
      }
      if (openRetryTimer !== null) {
        window.clearTimeout(openRetryTimer)
      }
      if (openRaf !== null) {
        window.cancelAnimationFrame(openRaf)
      }
      isIntentionalSocketClose = true
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close(1000, "Terminal instance disposed")
      } else if (ws && ws.readyState === WebSocket.CONNECTING) {
        ws.onopen = () => {
          ws.close(1000, "Terminal instance disposed before open")
        }
        ws.onmessage = null
        ws.onerror = null
        ws.onclose = null
      }
      xterm.dispose()
      isTerminalOpened = false
      xtermRef.current = null
      wsRef.current = null
    }
  }, [appId, instanceName, containerName, targetType])

  return (
    <div
      ref={terminalRef}
      className="h-full min-h-0 w-full overflow-hidden [&_.xterm]:h-full [&_.xterm]:w-full [&_.xterm-viewport]:overflow-y-auto!"
    />
  )
}
