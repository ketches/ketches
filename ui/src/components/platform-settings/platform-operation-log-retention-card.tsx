import { operationLogsApi } from "@/api/operation-logs"
import { OperationLogRetentionDialog } from "@/components/platform-settings/operation-log-retention-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useQuery } from "@tanstack/react-query"
import { Edit2, History } from "lucide-react"
import * as React from "react"

export function PlatformOperationLogRetentionCard() {
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const { data, isLoading } = useQuery({
    queryKey: ["operation-log-settings"],
    queryFn: () => operationLogsApi.getOperationLogSettings(),
  })

  const retentionDays = data?.retention_days ?? 90

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent group/card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-4 w-4" />
            Operation Log Retention
          </CardTitle>
          <CardDescription>
            Configure how long operation logs are retained before older entries are removed.
          </CardDescription>
          <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Edit retention days"
              disabled={isLoading}
              onClick={() => setDialogOpen(true)}
            >
              <Edit2 />
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="grid gap-1">
            <p className="text-[11px] font-medium text-muted-foreground">Current Retention</p>
            <p className="text-sm font-medium">{retentionDays} days</p>
          </div>
        </CardContent>
      </Card>

      <OperationLogRetentionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        retentionDays={retentionDays}
      />
    </>
  )
}
