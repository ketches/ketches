import {
  Download,
  FileClock,
  Pause,
  Play,
  RefreshCw,
  Settings2,
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { hasPersistedAuthSession } from "@/lib/auth-session"
import { cn } from "@/lib/utils"

import { WorkloadPanelFrame } from "./workload-panel-frame"

interface LogPanelProps {
  appId: string
  instanceName: string
  containerName: string
}

export function LogPanel({ appId, instanceName, containerName }: LogPanelProps) {
  const [logs, setLogs] = React.useState<string[]>([])
  const [tailLines, setTailLines] = React.useState(500)
  const [autoRefresh, setAutoRefresh] = React.useState(true)
  const [showTimestamp, setShowTimestamp] = React.useState(false)
  const [searchText, setSearchText] = React.useState("")
  const logContainerRef = React.useRef<HTMLDivElement>(null)
  const followTailRef = React.useRef(true)

  const isNearBottom = React.useCallback((container: HTMLDivElement) => {
    const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight
    return distanceToBottom <= 24
  }, [])

  const scheduleScrollToBottom = React.useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const container = logContainerRef.current
        if (!container || !autoRefresh || !followTailRef.current) return
        container.scrollTop = container.scrollHeight
      })
    })
  }, [autoRefresh])

  const handleContainerScroll = React.useCallback(() => {
    const container = logContainerRef.current
    if (!container) return
    followTailRef.current = isNearBottom(container)
  }, [isNearBottom])

  React.useEffect(() => {
    if (!appId || !instanceName || !containerName) return
    followTailRef.current = true

    if (!hasPersistedAuthSession()) {
      toast.error("Authentication required")
      return
    }

    const eventSource = new EventSource(
      `/api/v1/apps/${appId}/instances/${instanceName}/logs?container=${containerName}&tailLines=${tailLines}&timestamps=${showTimestamp}`,
      { withCredentials: true }
    )

    eventSource.onopen = () => {
      setLogs([])
      followTailRef.current = true
      scheduleScrollToBottom()
    }

    eventSource.onmessage = (event) => {
      if (autoRefresh) {
        setLogs((prev) => {
          const newLogs = [...prev, event.data]
          if (newLogs.length > 10000) {
            return newLogs.slice(newLogs.length - 5000)
          }
          return newLogs
        })
      }
    }

    eventSource.onerror = () => {
      toast.error("Failed to connect to log stream")
      eventSource.close()
    }

    return () => eventSource.close()
  }, [appId, instanceName, containerName, tailLines, showTimestamp, autoRefresh, isNearBottom, scheduleScrollToBottom])

  React.useLayoutEffect(() => {
    if (!autoRefresh || !followTailRef.current) return

    scheduleScrollToBottom()
  }, [logs, autoRefresh, scheduleScrollToBottom])

  React.useEffect(() => {
    const container = logContainerRef.current
    if (!container) return

    const observer = new ResizeObserver(() => {
      if (!autoRefresh || !followTailRef.current || logs.length === 0) return
      scheduleScrollToBottom()
    })
    observer.observe(container)

    return () => observer.disconnect()
  }, [autoRefresh, logs.length, scheduleScrollToBottom])

  const formatLogLine = (line: string) => {
    return line
  }

  const handleRefresh = () => {
    setLogs([])
  }

  const handleDownload = () => {
    const content = logs.join("\n")
    const blob = new Blob([content], { type: "text/plain" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `${instanceName}-${containerName}-logs.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success("Logs downloaded")
  }

  const filteredLogs = React.useMemo(() => {
    if (!searchText) return logs
    return logs.filter(log =>
      log.toLowerCase().includes(searchText.toLowerCase())
    )
  }, [logs, searchText])

  const highlightSearch = (line: string) => {
    if (!searchText) return formatLogLine(line)
    const formattedLine = formatLogLine(line)
    const regex = new RegExp(`(${searchText})`, "gi")
    return formattedLine.split(regex).map((part, i) =>
      regex.test(part) ? (
        <span key={i} className="bg-yellow-500/40 text-yellow-200 rounded px-0.5">
          {part}
        </span>
      ) : part
    )
  }

  return (
    <WorkloadPanelFrame
      toolbar={(
        <>
          <div className="flex items-center gap-2">
            <Input
              placeholder="Search logs..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              className="h-6 w-50 text-xs"
            />
            {searchText && (
              <span className="text-xs text-muted-foreground">
                {filteredLogs.length} / {logs.length} lines
              </span>
            )}
          </div>

          <div className="flex items-center gap-1">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={<Button variant="ghost" size="icon-sm" onClick={handleRefresh} />}
              >
                <RefreshCw className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Refresh</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={<Button variant="ghost" size="icon-sm" onClick={() => setAutoRefresh(!autoRefresh)} />}
              >
                {autoRefresh ? (
                  <Pause className="h-3.5 w-3.5" />
                ) : (
                  <Play className="h-3.5 w-3.5" />
                )}
              </TooltipTrigger>
              <TooltipContent>{autoRefresh ? "Pause" : "Resume"}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={handleDownload}
                    disabled={logs.length === 0}
                  />
                }
              >
                <Download className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Download logs</TooltipContent>
            </Tooltip>
            <DropdownMenu>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />} />}
                >
                  <Settings2 className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Log settings</TooltipContent>
              </Tooltip>
              <DropdownMenuContent align="end" className="w-50">
                <DropdownMenuGroup>
                  <DropdownMenuLabel className="text-xs">Settings</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <div className="p-2">
                    <Field>
                      <FieldLabel htmlFor="tailLines" className="text-xs">Initial Lines</FieldLabel>
                      <FieldContent>
                        <Input
                          id="tailLines"
                          type="number"
                          value={tailLines}
                          onChange={(e) => setTailLines(parseInt(e.target.value) || 500)}
                          min={100}
                          max={5000}
                          step={100}
                          className="h-7 text-xs"
                        />
                      </FieldContent>
                    </Field>
                  </div>
                  <DropdownMenuCheckboxItem
                    checked={showTimestamp}
                    onCheckedChange={setShowTimestamp}
                    className="text-xs"
                  >
                    Show Timestamp
                  </DropdownMenuCheckboxItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </>
      )}
      status={(
        <>
          <div className="flex items-center gap-4">
            <span>Lines: {logs.length}</span>
            <span className={cn(
              "flex items-center gap-1.5",
              autoRefresh ? "text-emerald-500" : "text-amber-500"
            )}>
              <span className={cn(
                "h-1.5 w-1.5 rounded-full",
                autoRefresh ? "bg-emerald-500 animate-pulse" : "bg-amber-500"
              )} />
              {autoRefresh ? "Live" : "Paused"}
            </span>
          </div>
          <div>Container: {containerName}</div>
        </>
      )}
    >

      <div
        ref={logContainerRef}
        onScroll={handleContainerScroll}
        className="h-full overflow-y-auto bg-zinc-950 p-3 font-mono text-xs text-zinc-100"
      >
        {filteredLogs.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-zinc-500">
            <FileClock className="h-10 w-10 mb-3 opacity-40" />
            <p className="text-sm">Waiting for logs...</p>
            {!autoRefresh && (
              <p className="text-xs mt-1.5 text-zinc-600">
                Auto refresh is paused. Click play to resume.
              </p>
            )}
          </div>
        ) : (
          <>
            {filteredLogs.map((log, i) => (
              <div
                key={i}
                className={cn(
                  "whitespace-pre-wrap break-all py-0.5 leading-relaxed hover:bg-zinc-900/50 px-1 -mx-1 rounded",
                  log.toLowerCase().includes("error") && "text-red-400",
                  log.toLowerCase().includes("warn") && "text-yellow-400"
                )}
              >
                <span className="text-zinc-600 select-none mr-3 inline-block w-8 text-right">
                  {i + 1}
                </span>
                {highlightSearch(log)}
              </div>
            ))}
          </>
        )}
      </div>
    </WorkloadPanelFrame>
  )
}
