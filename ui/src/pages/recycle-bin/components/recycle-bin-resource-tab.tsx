import { type LucideIcon, Loader2 } from "lucide-react"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import type { Dispatch, ReactNode, SetStateAction } from "react"

import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"

type RowSelectionState = Record<string, boolean>

interface BatchActionTable {
  getFilteredSelectedRowModel: () => {
    rows: unknown[]
  }
}

interface RecycleBinResourceTabProps<T> {
  title: string
  description: string
  icon: LucideIcon
  columns: ColumnDef<T>[]
  data: T[]
  isLoading: boolean
  isFetching: boolean
  leftToolbar: () => ReactNode
  batchActions?: (table: BatchActionTable) => ReactNode
  rowSelection: RowSelectionState
  onRowSelectionChange: Dispatch<SetStateAction<RowSelectionState>>
  onRefresh: () => Promise<unknown>
  totalCount: number
  pagination: PaginationState
  onPaginationChange: Dispatch<SetStateAction<PaginationState>>
}

export function RecycleBinResourceTab<T>({
  title,
  description,
  icon,
  columns,
  data,
  isLoading,
  isFetching,
  leftToolbar,
  batchActions,
  rowSelection,
  onRowSelectionChange,
  onRefresh,
  totalCount,
  pagination,
  onPaginationChange,
}: RecycleBinResourceTabProps<T>) {
  if (isLoading && data.length === 0) {
    return (
      <div className="flex min-h-100 flex-1 flex-col items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!isLoading && data.length === 0) {
    return (
      <EmptyState
        title={title}
        description={description}
        icon={icon}
      />
    )
  }

  return (
    <DataTable
      columns={columns}
      data={data}
      isLoading={isLoading || isFetching}
      leftToolbar={leftToolbar}
      batchActions={batchActions}
      rowSelection={rowSelection}
      onRowSelectionChange={onRowSelectionChange}
      onRefresh={onRefresh}
      manualPagination
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
    />
  )
}
