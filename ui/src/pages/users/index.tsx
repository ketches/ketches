import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Search, Trash2, User } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { usersApi, type ListUsersResponse, type User as UserType } from "@/api/users"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { AddUserDialog } from "@/components/users/add-user-dialog"
import { EditPasswordDialog } from "@/components/users/edit-password-dialog"
import { ImportUsersDialog } from "@/components/users/import-users-dialog"

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

const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
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

  const { data, isLoading, refetch } = useQuery<ListUsersResponse>({
    queryKey: ['users', pagination.pageIndex, pagination.pageSize, search],
    queryFn: () => usersApi.list({ page: pagination.pageIndex + 1, pageSize: pagination.pageSize, search }),
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
          <Select
            value={isValidRole ? user.role : "user"}
            onValueChange={(val) => val && updateRoleMutation.mutate({ userId: user.id, role: val })}
            disabled={updateRoleMutation.isPending}
          >
            <SelectTrigger className="h-8 w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {USER_ROLES.map((r) => (
                <SelectItem key={r} value={r}>
                  <div className="flex flex-col">
                    <span>{UserRoleLabels[r as UserRole]}</span>
                    <span className="text-xs text-muted-foreground">
                      {UserRoleDescriptions[r as UserRole]}
                    </span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )
      },
    },
    {
      accessorKey: "created_at",
      header: "Registered At",
      cell: ({ row }) => {
        return (
          <span className="text-muted-foreground">
            {formatDate(row.original.created_at ?? '')}
          </span>
        )
      },
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
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => deleteUserMutation.mutate(user.id)}
              disabled={deleteUserMutation.isPending || user.role === 'admin'}
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
            >
              <Trash2 />
              <span className="sr-only">Delete</span>
            </Button>
          </div>
        )
      },
    },
  ]

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <span className="text-muted-foreground animate-pulse">Loading users...</span>
      </div>
    )
  }

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

      {/* Search Bar */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search by username, email, or fullname..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      <DataTable
        columns={columns}
        data={users}
        onRefresh={refetch}
        manualPagination
        totalCount={total}
        pagination={pagination}
        onPaginationChange={setPagination}
        toolbarActions={(table) => {
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
                  Delete Selected ({selectedRows.length})
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
