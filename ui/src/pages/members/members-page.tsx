import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Trash2, Users } from "lucide-react"
import { toast } from "sonner"

import { PROJECT_ROLES, ProjectRole, ProjectRoleLabels, projectsApi, type ProjectMember } from "@/api/projects"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { AddMemberDialog } from "@/components/members/add-member-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useProjectStore } from "@/stores/project"

const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

export function MembersPage() {
  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: response, isLoading, refetch } = useQuery({
    queryKey: ['project-members', activeProjectId, pagination.pageIndex, pagination.pageSize],
    queryFn: () => projectsApi.listMembers(activeProjectId!, {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize
    }),
    enabled: !!activeProjectId,
  })

  const members = response?.items ?? []
  const paginationInfo = response?.pagination

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      projectsApi.addProjectMember(activeProjectId!, { user_id: userId, role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-members', activeProjectId] })
      toast.success("Member role updated")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update role",
      })
    }
  })

  const removeMemberMutation = useMutation({
    mutationFn: (userId: string) =>
      projectsApi.removeProjectMember(activeProjectId!, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-members', activeProjectId] })
      toast.success("Member removed from project")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to remove member",
      })
    }
  })

  const columns: ColumnDef<ProjectMember>[] = [
    {
      id: "select",
      size: 40,
      header: ({ table }) => (
        <div className="flex items-center px-1">
          <Checkbox
            checked={
              (table.getIsAllPageRowsSelected() ||
                (table.getIsSomePageRowsSelected() ? "mixed" : false)) as any
            }
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className="flex items-center px-1">
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "username",
      header: "Member",
      cell: ({ row }) => {
        const member = row.original
        return (
          <div className="flex items-center gap-3">
            <Avatar className="h-8 w-8 rounded-lg bg-primary/10 text-primary border-none">
              <AvatarFallback className="rounded-lg text-xs font-bold">
                {member.username.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div className="flex flex-col">
              <span className="font-medium text-foreground">{member.username}</span>
              <span className="text-xs text-muted-foreground">{member.email}</span>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "project_role",
      header: "Role",
      cell: ({ row }) => {
        const member = row.original
        return (
          <Select
            value={member.project_role}
            onValueChange={(val) => val && updateRoleMutation.mutate({ userId: member.user_id, role: val })}
            disabled={updateRoleMutation.isPending}
          >
            <SelectTrigger className="h-8 w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PROJECT_ROLES.map((r) => (
                <SelectItem key={r} value={r}>
                  {ProjectRoleLabels[r as ProjectRole]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )
      },
    },
    {
      accessorKey: "joined_at",
      header: "Joined At",
      cell: ({ row }) => {
        return <span className="text-muted-foreground">{formatDate(row.original.joined_at)}</span>
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const member = row.original
        return (
          <div className="flex justify-end gap-2">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => removeMemberMutation.mutate(member.user_id)}
              disabled={removeMemberMutation.isPending}
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
            >
              <Trash2 />
              <span className="sr-only">Remove</span>
            </Button>
          </div>
        )
      },
    },
  ]

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <span className="text-muted-foreground animate-pulse">Loading members...</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader items={[{ label: "Members", icon: Users }]} />
      <div className="flex flex-col gap-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Members</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage your project members and their roles.
            </p>
          </div>
        </div>

        <DataTable
          columns={columns}
          data={members}
          onRefresh={refetch}
          manualPagination
          totalCount={paginationInfo?.total || 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          searchKey="username"
          searchPlaceholder="Search members..."
          toolbarActions={(table) => {
            const selectedRows = table.getFilteredSelectedRowModel().rows
            return (
              <div className="flex items-center gap-2">
                {selectedRows.length > 0 && (
                  <Button
                    variant="destructive"
                    onClick={() => {
                      const ids = selectedRows.map((r) => r.original.user_id)
                      ids.forEach(id => removeMemberMutation.mutate(id))
                      table.resetRowSelection()
                    }}
                  >
                    <Trash2 />
                    Remove Selected ({selectedRows.length})
                  </Button>
                )}
                <AddMemberDialog
                  onAdd={(data) => {
                    updateRoleMutation.mutate({ userId: data.userId, role: data.role })
                  }}
                />
              </div>
            )
          }}
        />
      </div>
    </div>
  )
}

export default MembersPage
