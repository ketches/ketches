"use client"

import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type OnChangeFn,
  type RowSelectionState,
  type SortingState,
  type Table as TanstackTable,
  type VisibilityState,
} from "@tanstack/react-table"
import * as React from "react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Info, RefreshCw } from "lucide-react"

import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  searchKey?: string
  searchPlaceholder?: string
  toolbarActions?: (table: TanstackTable<TData>) => React.ReactNode
  leftActions?: (table: TanstackTable<TData>) => React.ReactNode
  batchActions?: (table: TanstackTable<TData>) => React.ReactNode
  onRowClick?: (data: TData) => void
  onRefresh?: () => void
  hidePagination?: boolean
  getRowClassName?: (data: TData) => string
  renderCard?: (data: TData, table: TanstackTable<TData>) => React.ReactNode
  viewMode?: "list" | "card"
  borderless?: boolean
  // Custom empty state rendered in place of "No results." when data is empty
  emptyContent?: React.ReactNode
  // Pagination
  pagination?: PaginationState
  onPaginationChange?: OnChangeFn<PaginationState>
  totalCount?: number
  manualPagination?: boolean
  // Controlled row selection
  rowSelection?: RowSelectionState
  onRowSelectionChange?: OnChangeFn<RowSelectionState>
}

import { EmptyState } from "@/components/shared/empty-state"
import { type PaginationState } from "@tanstack/react-table"

export function DataTable<TData, TValue>({
  columns,
  data,
  searchKey,
  searchPlaceholder = "Filter...",
  toolbarActions,
  leftActions,
  batchActions,
  onRowClick,
  onRefresh,
  hidePagination = false,
  getRowClassName,
  renderCard,
  viewMode = "list",
  borderless = true,
  emptyContent,
  pagination: paginationProp,
  onPaginationChange: onPaginationChangeProp,
  totalCount,
  manualPagination = false,
  rowSelection: controlledRowSelection,
  onRowSelectionChange,
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>([])
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    []
  )
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({})
  const [internalRowSelection, setInternalRowSelection] = React.useState({})
  const [internalPagination, setInternalPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  // Go-to-page input state
  const [goToPage, setGoToPage] = React.useState("")

  const rowSelection = controlledRowSelection ?? internalRowSelection
  const handleRowSelectionChange = onRowSelectionChange ?? setInternalRowSelection

  const pagination = paginationProp ?? internalPagination
  pagination.pageSize = viewMode === "card" ? 9 : 10
  const onPaginationChange = onPaginationChangeProp ?? setInternalPagination

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: manualPagination ? undefined : getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: handleRowSelectionChange,
    onPaginationChange,
    manualPagination: manualPagination,
    pageCount: manualPagination && totalCount !== undefined ? Math.ceil(totalCount / (pagination.pageSize || 10)) : undefined,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
      pagination,
    },
  })

  // Determine the minimum page size threshold for the current view mode
  const minPageSize = viewMode === "card" ? 9 : 10
  // For manual pagination use totalCount; for local pagination use data length
  const effectiveTotal = manualPagination ? (totalCount ?? 0) : data.length
  // Show pagination only when data exceeds the minimum page size
  const shouldShowPagination = !hidePagination && effectiveTotal > minPageSize

  // Handle go-to-page navigation
  const handleGoToPage = () => {
    const page = parseInt(goToPage, 10)
    if (!isNaN(page) && page >= 1 && page <= table.getPageCount()) {
      table.setPageIndex(page - 1)
    }
    setGoToPage("")
  }

  // Card mode: select-all checkbox and selected count shown in toolbar
  // Card mode: select-all shown on DataTable hover or when rows are selected
  const hasSelection = table.getFilteredSelectedRowModel().rows.length > 0
  const cardSelectAll = viewMode === "card" ? (
    <label className={cn(
      "flex items-center gap-2 cursor-pointer transition-opacity",
      hasSelection ? "opacity-100" : "opacity-0 group-hover/datatable:opacity-100"
    )}>
      <Checkbox
        checked={
          (table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() ? "mixed" : false)) as any
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
      />
      <p className="text-xs font-medium text-muted-foreground">Select all</p>
    </label>
  ) : null

  const selectedCountBadge = hasSelection ? (
    <i className="text-xs text-muted-foreground">
      {table.getFilteredSelectedRowModel().rows.length} of{" "}
      {table.getFilteredRowModel().rows.length} row(s) selected.
    </i>
  ) : null

  // Determine whether toolbar should be rendered
  const hasToolbar = searchKey || toolbarActions || batchActions || leftActions || viewMode === "card"

  return (
    <div className="group/datatable space-y-4">
      {hasToolbar && (
        <div className="flex items-center justify-between gap-4">
          <div className="flex flex-1 items-center gap-2">
            {searchKey && (
              <Input
                placeholder={searchPlaceholder}
                value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""}
                onChange={(event) =>
                  table.getColumn(searchKey)?.setFilterValue(event.target.value)
                }
                className="flex flex-1 max-w-sm min-w-75"
              />
            )}
            {/* leftActions: shown after search box (or alone when no search box) */}
            {leftActions?.(table)}
            {/* Card mode: select-all and selected count after search box */}
            {viewMode === "card" && cardSelectAll}
            {viewMode === "card" && selectedCountBadge}
          </div>
          <div className="flex items-center gap-2">
            {batchActions?.(table)}
            {onRefresh && (
              <Button
                variant="secondary"
                onClick={onRefresh}
                title="Refresh"
              >
                <RefreshCw />
              </Button>
            )}
            {toolbarActions?.(table)}
          </div>
        </div>
      )}
      {viewMode === "list" ? (
        <>
          {table.getRowModel().rows?.length ? (
            <div className={cn("rounded-md border", borderless && "border-x-0 rounded-none")}>
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow key={headerGroup.id}>
                      {headerGroup.headers.map((header) => {
                        return (
                          <TableHead key={header.id}>
                            {header.isPlaceholder
                              ? null
                              : flexRender(
                                header.column.columnDef.header,
                                header.getContext()
                              )}
                          </TableHead>
                        )
                      })}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {
                    table.getRowModel().rows.map((row) => (
                      <TableRow
                        key={row.id}
                        data-state={row.getIsSelected() && "selected"}
                        onClick={() => onRowClick?.(row.original)}
                        className={cn(
                          onRowClick ? "cursor-pointer" : "",
                          getRowClassName?.(row.original)
                        )}
                      >
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id}>
                            {flexRender(
                              cell.column.columnDef.cell,
                              cell.getContext()
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="col-span-full flex items-center">
              <EmptyState title="" description="No results." icon={Info} />
            </div>
          )
          }
        </>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <div key={row.id} className="relative group">
                <div className={cn(
                  "absolute top-2 left-2 z-10 transition-opacity",
                  row.getIsSelected() ? "opacity-100" : "opacity-0 group-hover:opacity-100",
                  table.getFilteredSelectedRowModel().rows.length > 0 ? "opacity-100" : ""
                )}>
                  <Checkbox
                    checked={row.getIsSelected()}
                    onCheckedChange={(value) => row.toggleSelected(!!value)}
                    aria-label="Select row"
                    className="bg-background"
                  />
                </div>
                {renderCard?.(row.original, table)}
              </div>
            ))
          ) : (
            emptyContent ? (
              <div className="col-span-full">
                {emptyContent}
              </div>
            ) : (
              <div className="col-span-full flex items-center">
                <EmptyState title="" description="No results." icon={Info} />
              </div>
            )
          )}
        </div>
      )}

      {shouldShowPagination && (
        <div className="flex items-center justify-between">
          <div className="flex-1 flex items-center gap-2 text-xs">
            {/* List mode: selected count shown here (card mode shows it in toolbar) */}
            {viewMode === "list" && selectedCountBadge}
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center space-x-6 lg:space-x-8">
              <div className="flex items-center space-x-2">
                <p className="text-xs font-medium text-muted-foreground">Rows per page</p>
                <Combobox
                  value={`${table.getState().pagination.pageSize}`}
                  onValueChange={(value: string | null) => {
                    if (value) {
                      table.setPageSize(Number(value))
                    }
                  }}
                >
                  <ComboboxInput className="h-7 w-16" />
                  <ComboboxContent>
                    <ComboboxList>
                      {(viewMode === "card" ? [9, 15, 30, 45, 60] : [10, 20, 30, 40, 50]).map((pageSize) => (
                        <ComboboxItem key={pageSize} value={`${pageSize}`}>
                          {`${pageSize}`}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </div>
              <div className="flex w-fit items-center justify-center text-xs font-medium text-muted-foreground">
                Page {table.getState().pagination.pageIndex + 1} of{" "}
                {table.getPageCount()}
              </div>
              {/* Go-to-page input */}
              <div className="flex items-center space-x-2">
                <p className="text-xs font-medium text-muted-foreground">Go to</p>
                <Input
                  className="h-7 w-14 text-xs text-center"
                  value={goToPage}
                  onChange={(e) => setGoToPage(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleGoToPage()
                  }}
                  placeholder={`${table.getState().pagination.pageIndex + 1}`}
                />
                <Button
                  variant="outline"
                  onClick={handleGoToPage}
                  disabled={table.getPageCount() <= 1}
                >
                  Go
                </Button>
              </div>
              <div className="flex items-center space-x-2">
                <Button
                  variant="outline"
                  onClick={() => table.setPageIndex(0)}
                  disabled={!table.getCanPreviousPage()}
                >
                  <span className="sr-only">Go to first page</span>
                  <ChevronsLeft />
                </Button>
                <Button
                  variant="outline"
                  onClick={() => table.previousPage()}
                  disabled={!table.getCanPreviousPage()}
                >
                  <span className="sr-only">Go to previous page</span>
                  <ChevronLeft />
                </Button>
                <Button
                  variant="outline"
                  onClick={() => table.nextPage()}
                  disabled={!table.getCanNextPage()}
                >
                  <span className="sr-only">Go to next page</span>
                  <ChevronRight />
                </Button>
                <Button
                  variant="outline"
                  onClick={() => table.setPageIndex(table.getPageCount() - 1)}
                  disabled={!table.getCanNextPage()}
                >
                  <span className="sr-only">Go to last page</span>
                  <ChevronsRight />
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
