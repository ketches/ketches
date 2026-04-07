import { type LucideIcon } from "lucide-react"
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
  return (
    <DataTable
      columns={columns}
      data={data}
      sourceDataCount={totalCount}
      isLoading={isLoading || isFetching}
      sourceEmptyContent={(
        <EmptyState
          title={title}
          description={description}
          icon={icon}
        />
      )}
      useStandaloneEmptyState
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
