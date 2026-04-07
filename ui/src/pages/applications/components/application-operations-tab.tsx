import { type OperationLogItem } from "@/api/operation-logs"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatDate } from "@/lib/utils"
import { type ColumnDef, type OnChangeFn, type PaginationState } from "@tanstack/react-table"
import { Footprints } from "lucide-react"

interface ApplicationOperationsTabProps {
  items: OperationLogItem[]
  isLoading: boolean
  isFetching: boolean
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  totalCount: number
}

const operationLogsColumns: ColumnDef<OperationLogItem>[] = [
  {
    accessorKey: "created_at",
    header: "Time",
    cell: ({ row }) => (
      <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>
    ),
  },
  {
    accessorKey: "username",
    header: "User",
  },
  {
    accessorKey: "action",
    header: "Action",
    cell: ({ row }) => <span className="text-sm font-medium">{row.original.action}</span>,
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge variant={row.original.status === "success" ? "secondary" : "destructive"}>
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "sensitivity",
    header: "Sensitivity",
    cell: ({ row }) => <span className="text-xs uppercase text-muted-foreground">{row.original.sensitivity}</span>,
  },
  {
    accessorKey: "client_ip",
    header: "IP",
    cell: ({ row }) => <span className="font-mono text-xs">{row.original.client_ip || "-"}</span>,
  },
]

export function ApplicationOperationsTab({
  items,
  isLoading,
  isFetching,
  pagination,
  onPaginationChange,
  totalCount,
}: ApplicationOperationsTabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <Footprints className="h-4 w-4" />
          Operation Logs
        </CardTitle>
        <CardDescription>
          Track recent operations executed against this application.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={operationLogsColumns}
          data={items}
          sourceDataCount={totalCount}
          isLoading={isLoading || isFetching}
          sourceEmptyContent={(
            <EmptyState
              title="No operation logs"
              description="Operations for this application will appear here."
              icon={Footprints}
            />
          )}
          useStandaloneEmptyState
          pagination={pagination}
          onPaginationChange={onPaginationChange}
          totalCount={totalCount}
          manualPagination
        />
      </CardContent>
    </Card>
  )
}
