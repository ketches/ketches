import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Footprints } from "lucide-react"
import type { Dispatch, SetStateAction } from "react"

import type { OperationLogItem } from "@/api/operation-logs"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatDate } from "@/lib/utils"

interface CodeRepositoryOperationLogsSectionProps {
  operationLogs: OperationLogItem[]
  isLoading: boolean
  isFetching: boolean
  pagination: PaginationState
  onPaginationChange: Dispatch<SetStateAction<PaginationState>>
  totalCount: number
}

export function CodeRepositoryOperationLogsSection({
  operationLogs,
  isLoading,
  isFetching,
  pagination,
  onPaginationChange,
  totalCount,
}: CodeRepositoryOperationLogsSectionProps) {
  const columns: ColumnDef<OperationLogItem>[] = [
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>,
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

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Footprints className="h-4 w-4" />
          Operation Logs
        </CardTitle>
        <CardDescription>
          Track recent operations executed against this code repository.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          data={operationLogs}
          sourceDataCount={totalCount}
          isLoading={isLoading || isFetching}
          sourceEmptyContent={(
            <EmptyState
              title="No operation logs"
              description="Operations for this code repository will appear here."
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
