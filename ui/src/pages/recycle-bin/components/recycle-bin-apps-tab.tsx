import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ArchiveRestore, Box, Clock, Trash2 } from "lucide-react"
import type { Dispatch, ReactNode, SetStateAction } from "react"

import { type RecycleBinApp } from "@/api/recycle-bin"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { formatDate } from "@/lib/utils"
import { RecycleBinResourceTab } from "./recycle-bin-resource-tab"

type RowSelectionState = Record<string, boolean>

interface RecycleBinAppsTabProps {
  data: RecycleBinApp[]
  isLoading: boolean
  isFetching: boolean
  leftToolbar: () => ReactNode
  batchActions?: (table: { getFilteredSelectedRowModel: () => { rows: unknown[] } }) => ReactNode
  rowSelection: RowSelectionState
  onRowSelectionChange: Dispatch<SetStateAction<RowSelectionState>>
  onRefresh: () => Promise<unknown>
  totalCount: number
  pagination: PaginationState
  onPaginationChange: Dispatch<SetStateAction<PaginationState>>
  isViewer: boolean
  restoringItemId: string | null
  deletingItemId: string | null
  onRestoreSingle: (id: string) => void
  onDeleteSingle: (id: string) => void
}

export function RecycleBinAppsTab({
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
  isViewer,
  restoringItemId,
  deletingItemId,
  onRestoreSingle,
  onDeleteSingle,
}: RecycleBinAppsTabProps) {
  const columns: ColumnDef<RecycleBinApp>[] = [
    {
      accessorKey: "name",
      header: "Application",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="rounded-md bg-blue-500/10 p-1.5 text-blue-600 shrink-0">
            <Box className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-xs font-medium">{row.original.name}</p>
            <p className="truncate font-mono text-xs text-muted-foreground">{row.original.slug}</p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "env_name",
      header: "Environment",
    },
    {
      accessorKey: "project_name",
      header: "Project",
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.project_name}</div>
          <div className="text-sm text-muted-foreground">{row.original.project_slug}</div>
        </div>
      ),
    },
    {
      accessorKey: "app_type",
      header: "Type",
    },
    {
      accessorKey: "deleted_at",
      header: "Deleted At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.deleted_at)}</span>
        </div>
      ),
    },
  ]

  if (!isViewer) {
    columns.unshift({
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    })
    columns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(event) => {
                    event.stopPropagation()
                    onRestoreSingle(row.original.id)
                  }}
                  disabled={restoringItemId === row.original.id}
                />
              }
            >
              <ArchiveRestore />
            </TooltipTrigger>
            <TooltipContent>Restore application</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={(event) => {
                    event.stopPropagation()
                    onDeleteSingle(row.original.id)
                  }}
                  disabled={deletingItemId === row.original.id}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Permanently delete</TooltipContent>
          </Tooltip>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    })
  }

  return (
    <RecycleBinResourceTab
      title="No deleted applications"
      description="Deleted applications will appear here. You can restore or permanently delete them."
      icon={Box}
      columns={columns}
      data={data}
      isLoading={isLoading}
      isFetching={isFetching}
      leftToolbar={leftToolbar}
      batchActions={!isViewer ? batchActions : undefined}
      rowSelection={rowSelection}
      onRowSelectionChange={onRowSelectionChange}
      onRefresh={onRefresh}
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
    />
  )
}
