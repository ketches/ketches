import * as React from "react"

import { PROJECT_ROLES, ProjectRole, ProjectRoleDescriptions, ProjectRoleLabels, projectsApi, type ProjectMember } from "@/api/projects"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { InviteMemberDialog } from "@/components/members/invite-member-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList
} from "@/components/ui/combobox"
import { InputGroupAddon } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, Copy, GalleryVerticalEnd, Trash2, UserKey, Users } from "lucide-react"
import { toast } from "sonner"

export function MembersPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const queryClient = useQueryClient()
  const { activeProjectId: activeProjectIdFromStore, activeProjectName } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectNameToUse = activeProjectName
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: response, isLoading, refetch } = useQuery({
    queryKey: ['project-members', activeProjectId, pagination.pageIndex, pagination.pageSize],
    queryFn: () => projectsApi.listMembers(activeProjectId!, {
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize
    }),
    enabled: !!activeProjectId,
  })

  const members = response?.items ?? []
  const paginationInfo = response?.pagination

  const inviteMembersMutation = useMutation({
    mutationFn: ({ userIds, role }: { userIds: string[]; role: string }) =>
      projectsApi.inviteProjectMembers(activeProjectId!, { user_ids: userIds, role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-members', activeProjectId] })
      queryClient.invalidateQueries({ queryKey: ['invitable-users', activeProjectId] })
      toast.success("Members invited successfully")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to invite members",
      })
    }
  })

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      projectsApi.inviteProjectMembers(activeProjectId!, { user_ids: [userId], role }),
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
              <span className="font-medium text-foreground">{member.fullname || member.username}</span>
              <span className="text-xs text-muted-foreground">{member.username}</span>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "email",
      header: "Email",
      cell: ({ row }) => {
        const member = row.original
        return <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="font-mono">
            {member.email}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(member.email)
              toast.success("Email address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      },
    },
    {
      accessorKey: "project_role",
      header: "Role",
      cell: ({ row }) => {
        const member = row.original
        return isViewer ? (
          <span className="text-xs">{ProjectRoleLabels[member.project_role as ProjectRole]}</span>
        ) : (
          <Combobox
            value={member.project_role}
            onValueChange={(val: string | null) => val && updateRoleMutation.mutate({ userId: member.user_id, role: val })}
            disabled={updateRoleMutation.isPending}
            itemToStringLabel={(v) => ProjectRoleLabels[v as ProjectRole] ?? v ?? ""}
          >
            <ComboboxInput className="w-fit" >
              <InputGroupAddon>
                <UserKey />
              </InputGroupAddon>
            </ComboboxInput>
            <ComboboxContent alignOffset={-24} className="w-full">
              <ComboboxList>
                {PROJECT_ROLES.map((r) => (
                  <ComboboxItem key={r} value={r}>
                    <Item size="xs" className="p-0">
                      <ItemContent>
                        <ItemTitle>{ProjectRoleLabels[r as ProjectRole]}</ItemTitle>
                        <ItemDescription>{ProjectRoleDescriptions[r as ProjectRole]}</ItemDescription>
                      </ItemContent>
                    </Item>
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
        )
      },
    },
    {
      accessorKey: "joined_at",
      header: "Joined At",
      cell: ({ row }) => {
        return <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.joined_at)}</span>
        </div>
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const member = row.original
        return (
          <div className="flex justify-end gap-2">
            {!isViewer && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => removeMemberMutation.mutate(member.user_id)}
                      disabled={removeMemberMutation.isPending}
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                    />
                  }
                >
                  <Trash2 />
                  <span className="sr-only">Remove</span>
                </TooltipTrigger>
                <TooltipContent>Remove member</TooltipContent>
              </Tooltip>
            )}
          </div>
        )
      },
    },
  ]

  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    { label: projectNameToUse || "Projects", icon: GalleryVerticalEnd, href: `/projects/${activeProjectId}` },
  ] : []
  breadcrumbs.push({ label: "Members", icon: Users })

  return (
    <div className="flex flex-col gap-6">
      {!projectIdProp && <PageHeader items={breadcrumbs} />}
      <div className="flex flex-col gap-6">
        {!projectIdProp && (
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Members</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage your project members and their roles.
              </p>
            </div>
          </div>
        )}

        <DataTable
          columns={columns}
          data={members}
          isLoading={isLoading}
          onRefresh={refetch}
          manualPagination
          totalCount={paginationInfo?.total || 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          searchKey="username"
          searchPlaceholder="Search members..."
          rightToolbar={(table) => {
            const selectedRows = table.getFilteredSelectedRowModel().rows
            return (
              <div className="flex items-center gap-2">
                {selectedRows.length > 0 && !isViewer && (
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
                {!isViewer && (
                  <InviteMemberDialog
                    projectId={activeProjectId!}
                    onAdd={(data) => {
                      inviteMembersMutation.mutate(data)
                    }}
                  />
                )}
              </div>
            )
          }}
        />
      </div>
    </div>
  )
}

export default MembersPage
