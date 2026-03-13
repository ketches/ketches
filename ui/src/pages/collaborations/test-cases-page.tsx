import { collaborationApi, type TestCase } from "@/api/collaboration"
import { CreateTestCaseDialog, CreateTestRunDialog, DeleteTestCaseDialog, EditTestCaseDialog } from "@/components/collaboration/test-case-dialogs"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { MoreVertical, Pencil, Play, Plus, TestTube, Trash2 } from "lucide-react"
import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"

interface TestCasesPageProps {
  projectId?: string
  sprintId?: string
}

export default function TestCasesPage({ projectId: propProjectId, sprintId }: TestCasesPageProps) {
  const params = useParams<{ projectId: string }>()
  const projectId = propProjectId || params.projectId

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebounce(search, 300)

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [runOpen, setRunOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<TestCase | null>(null)

  const { data: response, isLoading, refetch } = useQuery({
    queryKey: ["test-cases", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, sprintId],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listTestCases(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
        sprint_id: sprintId,
      })
    },
    enabled: !!projectId,
  })

  const testCases = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const handleEdit = (item: TestCase) => {
    setSelectedItem(item)
    setEditOpen(true)
  }

  const handleDelete = (item: TestCase) => {
    setSelectedItem(item)
    setDeleteOpen(true)
  }

  const handleRun = (item: TestCase) => {
    setSelectedItem(item)
    setRunOpen(true)
  }

  const columns: ColumnDef<TestCase>[] = useMemo(() => [
    {
      accessorKey: "title",
      header: "Title",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="font-medium truncate max-w-[500px]">{row.original.title}</span>
          <span className="text-xs text-muted-foreground font-mono">{row.original.id.slice(0, 8)}</span>
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>,
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" onClick={(e) => e.stopPropagation()} />}>
                <MoreVertical />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleRun(item)}>
                  <Play />
                  Run Test
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleEdit(item)}>
                  <Pencil />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem variant="destructive" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => handleDelete(item)}>
                  <Trash2 className="mr-2 h-3.5 w-3.5" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      },
    },
  ], [])

  if (!projectId) return null

  return (
    <div className="flex flex-col h-full gap-4">
      {!propProjectId && <PageHeader items={[{ label: "Test Cases", icon: TestTube }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Test Cases</h1>
            <p className="text-sm text-muted-foreground">
              Manage test scenarios and execute test runs.
            </p>
          </div>
        </div>
      )}

      {!isLoading && testCases.length === 0 ? (
        <EmptyState
          title="No test cases found"
          description="Create your first test case to get started."
          icon={TestTube}
          actionText="Create Test Case"
          onAction={() => setCreateOpen(true)}
          actionIcon={Plus}
        />
      ) : (
        <DataTable
          columns={columns}
          data={testCases}
          isLoading={isLoading}
          manualPagination
          totalCount={totalCount}
          pagination={pagination}
          onPaginationChange={setPagination}
          onRefresh={() => refetch()}
          leftToolbar={() => (
            <Input
              className="max-w-xs"
              placeholder="Search..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          )}
          rightToolbar={() => (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              New Test Case
            </Button>
          )}
        />
      )}

      <CreateTestCaseDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
      />

      <EditTestCaseDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        testCase={selectedItem}
      />

      {selectedItem && (
        <>
          <DeleteTestCaseDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            projectId={projectId}
            testCaseId={selectedItem.id}
            onSuccess={() => {
              setDeleteOpen(false)
              setSelectedItem(null)
            }}
          />
          <CreateTestRunDialog
            open={runOpen}
            onOpenChange={setRunOpen}
            projectId={projectId}
            testCase={selectedItem}
          />
        </>
      )}
    </div>
  )
}
