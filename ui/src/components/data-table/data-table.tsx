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
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Info, Loader2 } from "lucide-react"

import { RefreshButtonIcon } from "@/components/data-table/refresh-indicator"
import { useRefreshAction } from "@/components/data-table/use-refresh-action"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[] | null | undefined
  sourceDataCount?: number
  searchKey?: string
  searchPlaceholder?: string
  rightToolbar?: (table: TanstackTable<TData>) => React.ReactNode
  leftToolbar?: (table: TanstackTable<TData>) => React.ReactNode
  batchActions?: (table: TanstackTable<TData>) => React.ReactNode
  onRowClick?: (data: TData) => void
  onRefresh?: () => void
  refreshState?: {
    isRefreshing: boolean
    handleRefresh: () => void | Promise<unknown>
  }
  hidePagination?: boolean
  getRowClassName?: (data: TData) => string
  renderCard?: (data: TData, table: TanstackTable<TData>) => React.ReactNode
  viewMode?: "list" | "card"
  borderless?: boolean
  isLoading?: boolean
  getRowId?: (originalRow: TData, index: number) => string
  // Custom empty state rendered in place of the default filtered-empty state
  emptyContent?: React.ReactNode
  // Standalone empty state rendered instead of the table when the source data is empty
  sourceEmptyContent?: React.ReactNode
  // Render a standalone empty state when source data is empty after loading
  useStandaloneEmptyState?: boolean
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
  sourceDataCount,
  searchKey,
  searchPlaceholder = "Filter...",
  rightToolbar: rightToolbar,
  leftToolbar: leftToolbar,
  batchActions,
  onRowClick,
  onRefresh,
  refreshState,
  hidePagination = false,
  getRowClassName,
  renderCard,
  viewMode = "list",
  borderless = true,
  isLoading = false,
  getRowId,
  emptyContent,
  sourceEmptyContent,
  useStandaloneEmptyState = false,
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
  const onPaginationChange = onPaginationChangeProp ?? setInternalPagination
  const normalizedData = Array.isArray(data) ? data : []
  const effectiveSourceDataCount = sourceDataCount ?? normalizedData.length

  const table = useReactTable({
    data: normalizedData,
    columns,
    getRowId,
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
  const effectiveTotal = manualPagination ? (totalCount ?? 0) : normalizedData.length
  // Show pagination only when data exceeds the minimum page size
  const shouldShowPagination = !hidePagination && effectiveTotal > minPageSize
  const showInitialLoadingState = isLoading && effectiveSourceDataCount === 0
  const showStandaloneEmptyState = useStandaloneEmptyState && !isLoading && effectiveSourceDataCount === 0

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
  const selectedCountBadge = hasSelection ? (
    <i className="text-xs text-muted-foreground">
      {table.getFilteredSelectedRowModel().rows.length} of{" "}
      {table.getFilteredRowModel().rows.length} row(s) selected.
    </i>
  ) : null
  const cardToolbarSelection = viewMode === "card" && batchActions ? (
    <div
      className={cn(
        "flex items-center gap-3 transition-opacity",
        hasSelection ? "opacity-100" : "opacity-0 group-hover/datatable:opacity-100"
      )}
    >
      <label className="flex items-center gap-2 cursor-pointer">
        <Checkbox
          checked={
            (table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() ? "mixed" : false)) as boolean | undefined
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        />
        <p className="text-xs font-medium text-muted-foreground">Select all</p>
      </label>
      {selectedCountBadge}
    </div>
  ) : null

  // Determine whether toolbar should be rendered
  const hasToolbar = searchKey || rightToolbar || batchActions || leftToolbar || viewMode === "card"
  const visibleColumnCount = Math.max(table.getVisibleLeafColumns().length, 1)
  const internalRefreshAction = useRefreshAction({
    onRefresh,
    isLoading,
  })
  const isRefreshing = refreshState?.isRefreshing ?? internalRefreshAction.isRefreshing
  const handleRefresh = refreshState?.handleRefresh ?? internalRefreshAction.handleRefresh
  const showLoadingOverlay = isLoading && effectiveSourceDataCount > 0

  const listEmptyRow = (
    <TableRow>
      <TableCell colSpan={visibleColumnCount}>
        {emptyContent ?? <EmptyState title="" description="No matching results." icon={Info} border={false} />}
      </TableCell>
    </TableRow>
  )

  const loadingSkeleton = (
    <div>
      <Skeleton
        className={cn(
          "w-full rounded-xl bg-muted/10",
          viewMode === "list" ? "h-44" : "h-52",
          borderless && "rounded-xl"
        )}
      />
    </div>
  )

  if (showInitialLoadingState) {
    return loadingSkeleton
  }

  if (showStandaloneEmptyState) {
    return (
      sourceEmptyContent ?? (
        emptyContent ?? <EmptyState title="" description="No data available." icon={Info} />
      )
    )
  }

  return (
    <div className="group/datatable space-y-4">
      {hasToolbar && (
        <div className="flex items-center justify-between gap-4">
          <div className="flex flex-1 items-center gap-2 min-w-0">
            <div className="flex flex-1 items-center gap-2 min-w-0">
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
              {leftToolbar?.(table)}
            </div>
            {cardToolbarSelection}
          </div>
          <div className="flex items-center gap-2">
            {batchActions?.(table)}
            {onRefresh && (
              <Button
                variant="secondary"
                onClick={handleRefresh}
                disabled={isRefreshing || isLoading}
              >
                <RefreshButtonIcon spinning={isRefreshing} />
              </Button>
            )}
            {rightToolbar?.(table)}
          </div>
        </div>
      )}
      {viewMode === "list" ? (
        <>
          <div className={cn("relative rounded-md border", borderless && "border-x-0 rounded-none")}>
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
                {table.getRowModel().rows?.length
                  ? table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && "selected"}
                      onClick={() => onRowClick?.(row.original)}
                      className={cn(
                        "group/card group/row",
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
                  ))
                  : listEmptyRow}
              </TableBody>
            </Table>
            {showLoadingOverlay && (
              <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/55 backdrop-blur-[1px]">
                <div className="flex items-center gap-2 rounded-md border bg-background/95 px-3 py-1.5 text-xs text-muted-foreground shadow-sm">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Loading...</span>
                </div>
              </div>
            )}
          </div>
        </>
      ) : (
        <div className="relative">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <div key={row.id} className="relative group">
                  {batchActions && (
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
                    </div>)}
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
                  <EmptyState title="" description="No matching results." icon={Info} />
                </div>
              )
            )}
          </div>
          {showLoadingOverlay && (
            <div className="absolute inset-0 z-20 flex items-center justify-center rounded-md bg-background/55 backdrop-blur-[1px]">
              <div className="flex items-center gap-2 rounded-md border bg-background/95 px-3 py-1.5 text-xs text-muted-foreground shadow-sm">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Loading...</span>
              </div>
            </div>
          )}
        </div>
      )}

      {shouldShowPagination && (
        <div className="flex items-center justify-between">
          <div className="flex-1 flex items-center gap-2 text-xs">
            {/* List mode: selected count shown here (card mode shows it in toolbar) */}
            {viewMode === "list" && selectedCountBadge}

            <div className="flex w-fit items-center justify-center text-xs font-medium text-muted-foreground">
              Total {effectiveTotal} {effectiveTotal === 1 ? "row" : "rows"}
            </div>
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
                  <ComboboxContent side="top">
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
