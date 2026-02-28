import { useQuery } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import * as React from "react"

import { PROJECT_ROLES, ProjectRole, ProjectRoleLabels } from "@/api/projects"
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
import { SimpleCombobox } from "@/components/ui/simple-combobox"

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
                <SimpleCombobox
                  value={userId}
                  onValueChange={(val) => setUserId(val || "")}
                  options={users.map((u) => ({ value: u.id, label: `${u.username} (${u.email})` }))}
                  placeholder="Select a user"
                  className="w-full"
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel htmlFor="role">Role</FieldLabel>
              <FieldContent>
                <SimpleCombobox
                  value={role}
                  onValueChange={(val) => val && setRole(val)}
                  options={PROJECT_ROLES.map((r) => ({ value: r, label: ProjectRoleLabels[r as ProjectRole] }))}
                  placeholder="Select a role"
                  className="w-full"
                />
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
