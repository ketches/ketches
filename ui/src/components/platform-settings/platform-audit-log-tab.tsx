import { useQuery } from "@tanstack/react-query"
import { History } from "lucide-react"
import * as React from "react"

import { operationLogsApi } from "@/api/operation-logs"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { formatDate, toTitleCase } from "@/lib/utils"

const PAGE_SIZE = 10

export function PlatformAuditLogTab() {
  const [page, setPage] = React.useState(1)

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["platform-settings", "audit-logs", page],
    queryFn: () => operationLogsApi.listPlatformAuditLogs({ page, page_size: PAGE_SIZE }),
  })

  const items = data?.items ?? []
  const pagination = data?.pagination
  const totalPages = pagination?.total_pages ?? 0

  return (
    <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <History className="h-4 w-4" />
          Audit Log
        </CardTitle>
        <CardDescription>
          Review recent platform-level admin operations, including branding updates, manual update
          checks, rollout target changes, and rollout submissions.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="rounded-lg border bg-muted/20 p-4 text-sm text-muted-foreground">
            Loading platform audit logs...
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            title="No platform audit logs"
            description="Platform-level admin actions will appear here once changes are made."
            icon={History}
            border={false}
          />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <div key={item.id} className="rounded-lg border bg-background/80 p-4">
                <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                  <div className="min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">{toTitleCase(item.action)}</span>
                      <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground">
                        {item.resource_type.replaceAll("_", " ")}
                      </span>
                      <span
                        className="rounded-full px-2 py-0.5 text-[10px] uppercase tracking-wide"
                        data-status={item.status}
                      >
                        {item.status}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground">{item.request_summary || "No summary provided."}</p>
                  </div>
                  <div className="shrink-0 text-right text-xs text-muted-foreground">
                    <div>{item.username || "system"}</div>
                    <div>{formatDate(item.created_at)}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {totalPages > 1 ? (
          <div className="flex items-center justify-between gap-2 border-t pt-4">
            <p className="text-xs text-muted-foreground">
              Page {page} of {totalPages}
            </p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setPage((currentPage) => currentPage - 1)}
                disabled={page <= 1 || isFetching}
              >
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setPage((currentPage) => currentPage + 1)}
                disabled={page >= totalPages || isFetching}
              >
                Next
              </Button>
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
