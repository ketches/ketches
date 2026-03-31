import { ChevronLeft, ChevronRight, MessageSquarePlus } from "lucide-react"
import * as React from "react"

import type { BuilderSession } from "@/api/builder-sessions"
import { Button } from "@/components/ui/button"

interface BuilderSessionHistoryRailProps {
  sessions: BuilderSession[]
  selectedSessionId?: string
  onNewConversation: () => void
  onSelectSession: (sessionId: string) => void
}

function getSessionLabel(session: BuilderSession): string {
  return session.title || session.summary || session.id.slice(0, 8)
}

export function BuilderSessionHistoryRail({ sessions, selectedSessionId, onNewConversation, onSelectSession }: BuilderSessionHistoryRailProps) {
  const [expanded, setExpanded] = React.useState(true)

  return (
    <aside
      data-testid="builder-session-history"
      className={`h-full min-h-0 shrink-0 self-stretch overflow-hidden border border-border/70 bg-background/90 ${expanded ? "w-72" : ""} rounded-lg transition-width duration-150 ease-out`}
    >
      <div className="flex h-full min-h-0 flex-col">
        <div
          data-testid="builder-session-history-header"
          className="border-b bg-muted/35 px-1.5 py-1.5 backdrop-blur supports-backdrop-filter:bg-background/75"
        >
          <Button
            type="button"
            variant={expanded ? "secondary" : "ghost"}
            data-testid="builder-session-history-new"
            className={expanded ? "h-8 w-full justify-start rounded-lg px-3 text-xs" : "h-8 w-full justify-center rounded-lg px-2"}
            onClick={onNewConversation}
            size={expanded ? "sm" : "icon"}
          >
            <MessageSquarePlus className="h-4 w-4 shrink-0" />
            {expanded && <span
              aria-hidden={!expanded}
              className={`overflow-hidden whitespace-nowrap transition-[max-width,opacity,margin] duration-150 ease-out ${expanded ? "ml-2 max-w-40 opacity-100" : "ml-0 max-w-0 opacity-0"}`}
            >
              New conversation
            </span>}
          </Button>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden px-1.5 py-2">
          {expanded ? (
            <div data-testid="builder-session-history-list" className="h-full min-h-0 overflow-y-auto pr-1">
              <div className="space-y-0.5">
                {sessions.length === 0 ? (
                  <div className="rounded-lg border border-dashed bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                    No conversations yet
                  </div>
                ) : (
                  sessions.map((session) => {
                    const isSelected = session.id === selectedSessionId

                    return (
                      <Button
                        key={session.id}
                        type="button"
                        variant={isSelected ? "secondary" : "ghost"}
                        data-testid={`builder-session-history-item-${session.id}`}
                        className={`h-auto w-full justify-start rounded-xl px-3 py-2 text-left ${isSelected
                          ? "border border-border/80 bg-accent text-foreground shadow-sm"
                          : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                          }`}
                        onClick={() => onSelectSession(session.id)}
                      >
                        <div className="flex min-w-0 flex-col items-start gap-0.5">
                          <span className="truncate text-sm font-medium text-foreground">{getSessionLabel(session)}</span>
                          <span className="truncate text-[11px] text-muted-foreground">
                            {new Date(session.last_activity_at).toLocaleString()}
                          </span>
                        </div>
                      </Button>
                    )
                  })
                )}
              </div>
            </div>
          ) : null}
        </div>

        <div
          data-testid="builder-session-history-footer"
          className="border-t bg-muted/25 px-1.5 py-1.5 backdrop-blur supports-backdrop-filter:bg-background/75"
        >
          <Button
            type="button"
            variant="ghost"
            data-testid="builder-session-history-toggle"
            className={expanded ? "h-8 w-full justify-start rounded-lg px-3 text-xs" : "h-8 w-full justify-center rounded-lg px-2"}
            onClick={() => setExpanded((current) => !current)}
            size={expanded ? "sm" : "icon"}
          >
            {expanded ? <ChevronLeft className="h-4 w-4 shrink-0" /> : <ChevronRight className="h-4 w-4 shrink-0" />}
            {expanded && <span
              aria-hidden={!expanded}
              className={`overflow-hidden whitespace-nowrap transition-[max-width,opacity,margin] duration-150 ease-out ${expanded ? "ml-2 max-w-40 opacity-100" : "ml-0 max-w-0 opacity-0"}`}
            >
              Hide conversations
            </span>}
          </Button>
        </div>
      </div>
    </aside>
  )
}
