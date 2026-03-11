import { usersApi, type ListUsersResponse, type User as UserType } from "@/api/users"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { AddUserDialog } from "@/components/users/add-user-dialog"
import { EditPasswordDialog } from "@/components/users/edit-password-dialog"
import { ImportUsersDialog } from "@/components/users/import-users-dialog"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, Trash2, User } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

const USER_ROLES = ["admin", "user"] as const
type UserRole = (typeof USER_ROLES)[number]

const UserRoleLabels: Record<UserRole, string> = {
  admin: "Admin",
  user: "User",
}

const UserRoleDescriptions: Record<UserRole, string> = {
  admin: "Full system access",
  user: "Regular user access",
}

export function UsersPage() {
  const queryClient = useQueryClient()

  // Pagination and search state
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [search, setSearch] = useState("")
  const [searchInput, setSearchInput] = useState("")

  // Debounced search
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput)
      setPagination(p => ({ ...p, pageIndex: 0 })) // Reset to first page on search
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput])

  const { data, refetch } = useQuery<ListUsersResponse>({
    queryKey: ['users', pagination.pageIndex, pagination.pageSize, search],
    queryFn: () => usersApi.list({ page: pagination.pageIndex + 1, page_size: pagination.pageSize, search }),
    placeholderData: (previousData) => previousData,
  })

  const users = data?.users ?? []
  const total = data?.total ?? 0

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      usersApi.updateRole(userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success("User role updated")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update role",
      })
    }
  })

  const updatePasswordMutation = useMutation({
    mutationFn: ({ userId, password }: { userId: string; password: string }) =>
      usersApi.updatePassword(userId, password),
    onSuccess: () => {
      toast.success("Password updated successfully")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update password",
      })
    }
  })

  const deleteUserMutation = useMutation({
    mutationFn: (userId: string) => usersApi.delete(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success("User deleted successfully")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to delete user",
      })
    }
  })

  const columns: ColumnDef<UserType>[] = [
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
      header: "User",
      cell: ({ row }) => {
        const user = row.original
        return (
          <div className="flex items-center gap-3">
            <Avatar className="h-8 w-8 rounded-lg bg-primary/10 text-primary border-none">
              <AvatarFallback className="rounded-lg text-xs font-bold">
                {user.username.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div className="flex flex-col">
              <span className="font-medium text-foreground">{user.fullname || user.username}</span>
              <span className="text-xs text-muted-foreground font-mono">{user.username || "-"}</span>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "email",
      header: "Email",
      cell: ({ row }) => {
        return <span className="text-muted-foreground">{row.original.email || "-"}</span>
      },
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) => {
        const user = row.original
        const isValidRole = USER_ROLES.includes(user.role as UserRole)

        return (
          <Combobox
            value={isValidRole ? user.role : "user"}
            onValueChange={(val: string | null) => val && updateRoleMutation.mutate({ userId: user.id, role: val })}
            disabled={updateRoleMutation.isPending}
            itemToStringLabel={(v) => UserRoleLabels[v as UserRole] ?? v ?? ""}
          >
            <ComboboxInput className="w-40" />
            <ComboboxContent>
              <ComboboxList>
                {USER_ROLES.map((r) => (
                  <ComboboxItem key={r} value={r}>
                    <Item size="xs" className="p-0">
                      <ItemContent>
                        <ItemTitle>{UserRoleLabels[r as UserRole]}</ItemTitle>
                        <ItemDescription>{UserRoleDescriptions[r as UserRole]}</ItemDescription>
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
      accessorKey: "created_at",
      header: "Registered At",
      cell: ({ row }) => {
        return (
          <div className="flex items-center gap-1.5 text-muted-foreground">
            <Clock className="h-3 w-3" />
            <span>{formatDate(row.original.created_at)}</span>
          </div>
        )
      }
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const user = row.original
        return (
          <div className="flex justify-end gap-2">
            <EditPasswordDialog
              userId={user.id}
              username={user.username}
              onSubmit={(password) => updatePasswordMutation.mutate({ userId: user.id, password })}
              isPending={updatePasswordMutation.isPending}
            />
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => deleteUserMutation.mutate(user.id)}
                    disabled={deleteUserMutation.isPending || user.role === 'admin'}
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  />
                }
              >
                <Trash2 />
                <span className="sr-only">Delete</span>
              </TooltipTrigger>
              <TooltipContent>Delete user</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
  ]



  return (
    <div className="flex flex-col gap-6">
      <PageHeader items={[{ label: "Users", icon: User }]} />
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Users</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage system users and their roles.
          </p>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={users}
        isLoading={!data}
        onRefresh={refetch}
        manualPagination
        totalCount={total}
        pagination={pagination}
        onPaginationChange={setPagination}
        leftToolbar={() => {
          return (
            <Input
              className="flex flex-1 max-w-sm min-w-75"
              placeholder="Search users..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
            />
          )
        }}
        rightToolbar={(table) => {
          const selectedRows = table.getFilteredSelectedRowModel().rows
          return (
            <div className="flex items-center gap-2">
              {selectedRows.length > 0 && (
                <Button
                  variant="destructive"
                  onClick={() => {
                    const ids = selectedRows.map((r) => r.original.id)
                    ids.forEach(id => deleteUserMutation.mutate(id))
                    table.resetRowSelection()
                  }}
                >
                  <Trash2 />
                  Delete
                </Button>
              )}
              <AddUserDialog onSuccess={() => refetch()} />
              <ImportUsersDialog onSuccess={() => refetch()} />
            </div>
          )
        }}
      />
    </div>
  )
}

export default UsersPage
