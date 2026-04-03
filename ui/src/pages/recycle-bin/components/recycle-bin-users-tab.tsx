import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ArchiveRestore, Clock, Trash2, User } from "lucide-react"
import type { Dispatch, ReactNode, SetStateAction } from "react"

import { type RecycleBinUser } from "@/api/recycle-bin"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { formatDate } from "@/lib/utils"
import { RecycleBinResourceTab } from "./recycle-bin-resource-tab"

type RowSelectionState = Record<string, boolean>

interface RecycleBinUsersTabProps {
  data: RecycleBinUser[]
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
  isAdmin: boolean
  restoringItemId: string | null
  deletingItemId: string | null
  onRestoreSingle: (id: string) => void
  onDeleteSingle: (id: string) => void
}

export function RecycleBinUsersTab({
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
  isAdmin,
  restoringItemId,
  deletingItemId,
  onRestoreSingle,
  onDeleteSingle,
}: RecycleBinUsersTabProps) {
  const columns: ColumnDef<RecycleBinUser>[] = [
    {
      accessorKey: "username",
      header: "User",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Avatar className="h-8 w-8 rounded-lg border-none bg-primary/10 text-primary">
            <AvatarFallback className="rounded-lg text-xs font-bold">
              {row.original.username.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col">
            <span className="font-medium text-foreground">{row.original.fullname || row.original.username}</span>
            <span className="font-mono text-xs text-muted-foreground">{row.original.username}</span>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "email",
      header: "Email",
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.email || "-"}</span>,
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) => <span className="capitalize">{row.original.role}</span>,
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

  if (isAdmin) {
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
            <TooltipContent>Restore user</TooltipContent>
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
      title="No deleted users"
      description="Deleted users will appear here. You can restore or permanently delete them."
      icon={User}
      columns={columns}
      data={data}
      isLoading={isLoading}
      isFetching={isFetching}
      leftToolbar={leftToolbar}
      batchActions={batchActions}
      rowSelection={rowSelection}
      onRowSelectionChange={onRowSelectionChange}
      onRefresh={onRefresh}
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
    />
  )
}
