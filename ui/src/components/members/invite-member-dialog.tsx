import { useQuery } from "@tanstack/react-query"
import { UserPlus } from "lucide-react"
import * as React from "react"

import { PROJECT_ROLES, ProjectRole, ProjectRoleDescriptions, ProjectRoleLabels, projectsApi } from "@/api/projects"
import { MemberAvatar } from "@/components/shared/member-avatar"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Field,
  FieldContent,
  FieldLabel
} from "@/components/ui/field"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"

interface InvitableUser {
  id: string
  username: string
  email: string
  fullname?: string
}

export function InviteMemberDialog({ projectId, onAdd }: { projectId: string; onAdd: (data: { userIds: string[], role: string }) => void }) {
  const [open, setOpen] = React.useState(false)
  const [userIds, setUserIds] = React.useState<string[]>([])
  const [role, setRole] = React.useState<string>(ProjectRole.DEVELOPER)
  const [searchQuery, setSearchQuery] = React.useState("")

  const { data: users = [] } = useQuery({
    queryKey: ['invitable-users', projectId],
    queryFn: () => projectsApi.listInvitableUsers(projectId),
    enabled: open && !!projectId,
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (userIds.length === 0) return
    onAdd({ userIds, role })
    setOpen(false)
    setUserIds([])
    setRole(ProjectRole.DEVELOPER)
    setSearchQuery("")
  }

  const filteredUsers = React.useMemo(() => {
    if (!searchQuery) return users
    const query = searchQuery.toLowerCase()
    return users.filter(
      (u) =>
        u.username.toLowerCase().includes(query) ||
        u.email.toLowerCase().includes(query)
    )
  }, [users, searchQuery])

  const anchor = useComboboxAnchor()

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <UserPlus />
        Invite Members
      </DialogTrigger>
      <DialogContent showCloseButton>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Invite Members</DialogTitle>
            <DialogDescription>
              Invite new members to your project and assign them a role.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="user">Users</FieldLabel>
              <FieldContent>
                <Combobox
                  value={userIds}
                  onValueChange={(val: string[] | null) => setUserIds(val || [])}
                  onInputValueChange={setSearchQuery}
                  multiple
                  items={filteredUsers}
                  itemToStringLabel={(u: string | InvitableUser) => (typeof u === 'string' ? (users.find(user => user.id === u)?.username ?? u) : u.username)}
                >
                  <ComboboxChips ref={anchor} className="w-full">
                    <ComboboxValue>
                      {(values: string[]) => (
                        <React.Fragment>
                          {values.map((id: string) => {
                            const user = users.find((u) => u.id === id)
                            if (!user) return null
                            return (
                              <ComboboxChip key={id}>
                                {user.fullname}
                              </ComboboxChip>
                            )
                          })}
                          <ComboboxChipsInput placeholder="Select users" />
                        </React.Fragment>
                      )}
                    </ComboboxValue>
                  </ComboboxChips>
                  <ComboboxContent anchor={anchor}>
                    <ComboboxEmpty>No users found.</ComboboxEmpty>
                    <ComboboxList>
                      {(u) => (
                        <ComboboxItem key={u.id} value={u.id}>
                          <Item size="xs" className="p-0">
                            <ItemContent className="grid grid-cols-[auto,1fr] gap-2 items-center">
                              <MemberAvatar name={u.fullname} />
                              <div className="min-w-0">
                                <ItemTitle>{u.fullname}</ItemTitle>
                                <ItemDescription className="line-clamp-1">
                                  {u.email}
                                </ItemDescription>
                              </div>
                            </ItemContent>
                          </Item>
                        </ComboboxItem>
                      )}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel htmlFor="role">Role</FieldLabel>
              <FieldContent>
                <Combobox
                  value={role}
                  onValueChange={(val: string | null) => val && setRole(val)}
                  itemToStringLabel={(v) => ProjectRoleLabels[v as ProjectRole] ?? v ?? ""}
                >
                  <ComboboxInput placeholder="Select a role" />
                  <ComboboxContent>
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
              </FieldContent>
            </Field>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={userIds.length === 0}>Add Member{userIds.length > 1 ? 's' : ''}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
