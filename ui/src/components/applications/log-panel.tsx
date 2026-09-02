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

interface LogEntry {
  id: number
  text: string
}

const LOG_BUFFER_CAPACITY = 10000
const LOG_FLUSH_INTERVAL_MS = 50
const LOG_ROW_HEIGHT = 24
const LOG_OVERSCAN_ROWS = 10
const DEFAULT_VIEWPORT_HEIGHT = 600

class LogRingBuffer {
  private readonly entries: Array<LogEntry | undefined>
  private readonly capacity: number
  private head = 0
  private size = 0

  constructor(capacity: number) {
    this.capacity = capacity
    this.entries = new Array(capacity)
  }

  append(entry: LogEntry) {
    if (this.size < this.capacity) {
      this.entries[(this.head + this.size) % this.capacity] = entry
      this.size += 1
      return
    }

    this.entries[this.head] = entry
    this.head = (this.head + 1) % this.capacity
  }

  clear() {
    this.head = 0
    this.size = 0
  }

  snapshot() {
    const result: LogEntry[] = []
    for (let index = 0; index < this.size; index += 1) {
      const entry = this.entries[(this.head + index) % this.capacity]
      if (entry) result.push(entry)
    }
    return result
  }
}

export function LogPanel({ appId, instanceName, containerName }: LogPanelProps) {
  const [logs, setLogs] = React.useState<LogEntry[]>([])
  const [tailLines, setTailLines] = React.useState(500)
  const [autoRefresh, setAutoRefresh] = React.useState(true)
  const [showTimestamp, setShowTimestamp] = React.useState(false)
  const [searchText, setSearchText] = React.useState("")
  const [scrollTop, setScrollTop] = React.useState(0)
  const [viewportHeight, setViewportHeight] = React.useState(0)
  const logContainerRef = React.useRef<HTMLDivElement>(null)
  const followTailRef = React.useRef(true)
  const autoRefreshRef = React.useRef(autoRefresh)
  const logBufferRef = React.useRef(new LogRingBuffer(LOG_BUFFER_CAPACITY))
  const pendingLogsRef = React.useRef(new LogRingBuffer(LOG_BUFFER_CAPACITY))
  const nextLogIdRef = React.useRef(1)
  const flushTimerRef = React.useRef<number | null>(null)

  React.useEffect(() => {
    autoRefreshRef.current = autoRefresh
  }, [autoRefresh])

  const isNearBottom = React.useCallback((container: HTMLDivElement) => {
    const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight
    return distanceToBottom <= 24
  }, [])

  const scheduleScrollToBottom = React.useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const container = logContainerRef.current
        if (!container || !autoRefreshRef.current || !followTailRef.current) return
        container.scrollTop = container.scrollHeight
        setScrollTop(container.scrollTop)
      })
    })
  }, [])

  const handleContainerScroll = React.useCallback(() => {
    const container = logContainerRef.current
    if (!container) return
    followTailRef.current = isNearBottom(container)
    setScrollTop(container.scrollTop)
  }, [isNearBottom])

  const flushPendingLogs = React.useCallback(() => {
    flushTimerRef.current = null
    const pendingLogs = pendingLogsRef.current.snapshot()
    if (pendingLogs.length === 0) return

    pendingLogsRef.current.clear()
    for (const entry of pendingLogs) {
      logBufferRef.current.append(entry)
    }
    setLogs(logBufferRef.current.snapshot())
  }, [])

  const scheduleLogFlush = React.useCallback(() => {
    if (flushTimerRef.current !== null) return
    flushTimerRef.current = window.setTimeout(flushPendingLogs, LOG_FLUSH_INTERVAL_MS)
  }, [flushPendingLogs])

  React.useEffect(() => {
    if (!appId || !instanceName || !containerName) return
    followTailRef.current = true
    const pendingLogs = pendingLogsRef.current

    if (!hasPersistedAuthSession()) {
      toast.error("Authentication required")
      return
    }

    const eventSource = new EventSource(
      `/api/v1/apps/${appId}/instances/${instanceName}/logs?container=${containerName}&tailLines=${tailLines}&timestamps=${showTimestamp}`,
      { withCredentials: true }
    )

    eventSource.onopen = () => {
      logBufferRef.current.clear()
      pendingLogs.clear()
      nextLogIdRef.current = 1
      setLogs([])
      followTailRef.current = true
      scheduleScrollToBottom()
    }

    eventSource.onmessage = (event) => {
      if (!autoRefreshRef.current) return
      pendingLogs.append({
        id: nextLogIdRef.current,
        text: event.data,
      })
      nextLogIdRef.current += 1
      scheduleLogFlush()
    }

    eventSource.onerror = () => {
      toast.error("Failed to connect to log stream")
      eventSource.close()
    }

    return () => {
      eventSource.close()
      pendingLogs.clear()
      if (flushTimerRef.current !== null) {
        window.clearTimeout(flushTimerRef.current)
        flushTimerRef.current = null
      }
    }
  }, [appId, instanceName, containerName, tailLines, showTimestamp, scheduleLogFlush, scheduleScrollToBottom])

  React.useLayoutEffect(() => {
    if (!autoRefresh || !followTailRef.current) return

    scheduleScrollToBottom()
  }, [logs, autoRefresh, scheduleScrollToBottom])

  React.useEffect(() => {
    const container = logContainerRef.current
    if (!container) return

    const updateViewportHeight = () => {
      setViewportHeight(container.clientHeight)
      if (!autoRefresh || !followTailRef.current || logs.length === 0) return
      scheduleScrollToBottom()
    }

    updateViewportHeight()
    const observer = new ResizeObserver(updateViewportHeight)
    observer.observe(container)

    return () => observer.disconnect()
  }, [autoRefresh, logs.length, scheduleScrollToBottom])

  const formatLogLine = (line: string) => {
    return line
  }

  const handleRefresh = () => {
    logBufferRef.current.clear()
    pendingLogsRef.current.clear()
    nextLogIdRef.current = 1
    setLogs([])
  }

  const handleDownload = () => {
    const content = logs.map((log) => log.text).join("\n")
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
    const normalizedSearch = searchText.toLowerCase()
    return logs.filter(log =>
      log.text.toLowerCase().includes(normalizedSearch)
    )
  }, [logs, searchText])

  const visibleRange = React.useMemo(() => {
    const measuredHeight = viewportHeight || DEFAULT_VIEWPORT_HEIGHT
    const start = Math.max(0, Math.floor(scrollTop / LOG_ROW_HEIGHT) - LOG_OVERSCAN_ROWS)
    const end = Math.min(
      filteredLogs.length,
      Math.ceil((scrollTop + measuredHeight) / LOG_ROW_HEIGHT) + LOG_OVERSCAN_ROWS
    )
    return { start, end }
  }, [filteredLogs.length, scrollTop, viewportHeight])

  const visibleLogs = React.useMemo(
    () => filteredLogs.slice(visibleRange.start, visibleRange.end),
    [filteredLogs, visibleRange]
  )

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(event.target.value)
    setScrollTop(0)
    if (logContainerRef.current) logContainerRef.current.scrollTop = 0
  }

  const handleAutoRefreshToggle = () => {
    setAutoRefresh((current) => {
      const next = !current
      autoRefreshRef.current = next
      return next
    })
  }

  const highlightSearch = (line: string) => {
    const formattedLine = formatLogLine(line)
    if (!searchText) return formattedLine

    const normalizedLine = formattedLine.toLowerCase()
    const normalizedSearch = searchText.toLowerCase()
    const parts: React.ReactNode[] = []
    let cursor = 0
    let matchIndex = normalizedLine.indexOf(normalizedSearch)

    while (matchIndex !== -1) {
      if (matchIndex > cursor) {
        parts.push(formattedLine.slice(cursor, matchIndex))
      }

      const matchEnd = matchIndex + searchText.length
      parts.push(
        <span
          key={`${matchIndex}-${matchEnd}`}
          className="bg-yellow-500/40 text-yellow-200 rounded px-0.5"
        >
          {formattedLine.slice(matchIndex, matchEnd)}
        </span>
      )
      cursor = matchEnd
      matchIndex = normalizedLine.indexOf(normalizedSearch, cursor)
    }

    if (cursor < formattedLine.length) {
      parts.push(formattedLine.slice(cursor))
    }

    return parts
  }

  return (
    <WorkloadPanelFrame
      toolbar={(
        <>
          <div className="flex items-center gap-2">
            <Input
              placeholder="Search logs..."
              value={searchText}
              onChange={handleSearchChange}
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
                render={<Button variant="ghost" size="icon-sm" onClick={handleAutoRefreshToggle} />}
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
          <div
            data-testid="virtual-log-list"
            className="min-w-full"
          >
            <svg
              aria-hidden="true"
              data-testid="virtual-log-spacer"
              className="block w-px"
              height={visibleRange.start * LOG_ROW_HEIGHT}
              width="1"
            />
            {visibleLogs.map((log, visibleIndex) => {
              const logIndex = visibleRange.start + visibleIndex
              const normalizedLog = log.text.toLowerCase()
              return (
                <div
                  key={log.id}
                  data-testid="virtual-log-row"
                  className={cn(
                    "h-6 whitespace-pre px-1 py-0.5 leading-relaxed hover:bg-zinc-900/50",
                    normalizedLog.includes("error") && "text-red-400",
                    normalizedLog.includes("warn") && "text-yellow-400"
                  )}
                >
                  <span className="text-zinc-600 select-none mr-3 inline-block w-8 text-right">
                    {logIndex + 1}
                  </span>
                  {highlightSearch(log.text)}
                </div>
              )
            })}
            <svg
              aria-hidden="true"
              data-testid="virtual-log-spacer"
              className="block w-px"
              height={(filteredLogs.length - visibleRange.end) * LOG_ROW_HEIGHT}
              width="1"
            />
          </div>
        )}
      </div>
    </WorkloadPanelFrame>
  )
}
