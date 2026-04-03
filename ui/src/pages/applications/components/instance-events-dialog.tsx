import { appsApi } from "@/api/apps"
import { DataTable } from "@/components/data-table/data-table"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { formatDate } from "@/lib/utils"
import { useQuery } from "@tanstack/react-query"
import { Activity } from "lucide-react"

interface InstanceEventsDialogProps {
  appId: string
  instanceName: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function InstanceEventsDialog({
  appId,
  instanceName,
  open,
  onOpenChange,
}: InstanceEventsDialogProps) {
  const { data: events = [], isLoading } = useQuery({
    queryKey: ["instance-events", appId, instanceName],
    queryFn: () => appsApi.listInstanceEvents(appId, instanceName),
    enabled: !!appId && !!instanceName && open,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-240 max-h-[80vh] max-h[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Instance Events: {instanceName}</DialogTitle>
          <DialogDescription>
            Recent events related to this instance, such as scaling actions, health status changes, and lifecycle events.
          </DialogDescription>
        </DialogHeader>
        <div className="flex-1 overflow-auto py-4">
          <DataTable
            columns={[
              {
                accessorKey: "type",
                header: "Type",
                cell: ({ row }) => (
                  <ColorBadge color={row.original.type === "Normal" ? "blue" : "red"}>
                    {row.original.type}
                  </ColorBadge>
                ),
              },
              { accessorKey: "reason", header: "Reason" },
              {
                accessorKey: "message",
                header: "Message",
                cell: ({ row }) => (
                  <div className="text-xs break-all whitespace-normal">
                    {row.original.message}
                  </div>
                ),
              },
              {
                accessorKey: "count",
                header: "Count",
                cell: ({ row }) => <span className="text-xs font-mono">{row.original.count}</span>,
              },
              {
                accessorKey: "created_at",
                header: "Last Seen",
                cell: ({ row }) => <span className="text-xs text-muted-foreground whitespace-nowrap">{formatDate(row.original.created_at)}</span>,
              },
            ]}
            data={events}
            isLoading={isLoading}
            hidePagination
            emptyContent={(
              <EmptyState
                title=""
                description="No events found for this instance."
                icon={Activity}
                border={false}
              />
            )}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
