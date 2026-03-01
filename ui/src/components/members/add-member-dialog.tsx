import { useQuery } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import * as React from "react"

import { PROJECT_ROLES, ProjectRole, ProjectRoleLabels, ProjectRoleDescriptions } from "@/api/projects"
import { usersApi } from "@/api/users"
import { Button } from "@/components/ui/button"
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
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Item, ItemContent, ItemTitle, ItemDescription } from "@/components/ui/item"



export function AddMemberDialog({ onAdd }: { onAdd: (data: { userId: string, role: string }) => void }) {
  const [open, setOpen] = React.useState(false)
  const [userId, setUserId] = React.useState("")
  const [role, setRole] = React.useState<string>(ProjectRole.DEVELOPER)

  const { data } = useQuery({
    queryKey: ['users'],
    queryFn: () => usersApi.list({}),
    enabled: open,
  })

  const users = data?.users ?? []

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!userId) return
    onAdd({ userId, role })
    setOpen(false)
    setUserId("")
    setRole(ProjectRole.DEVELOPER)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus />
        Add Member
      </DialogTrigger>
      <DialogContent showCloseButton>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Member</DialogTitle>
            <DialogDescription>
              Add a new member to your project and assign them a role.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="user">User</FieldLabel>
              <FieldContent>
                <Combobox
                  value={userId}
                  onValueChange={(val: string | null) => setUserId(val || "")}
                  itemToStringLabel={(id) => {
                    const u = users.find((user) => user.id === id)
                    return u ? `${u.username} (${u.email})` : id ?? ""
                  }}
                >
                    <ComboboxInput placeholder="Select a user" />
                  <ComboboxContent>
                    <ComboboxList>
                      {users.map((u) => (
                        <ComboboxItem key={u.id} value={u.id}>
                          {`${u.username} (${u.email})`}
                        </ComboboxItem>
                      ))}
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
            <Button type="submit" disabled={!userId}>Add Member</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
