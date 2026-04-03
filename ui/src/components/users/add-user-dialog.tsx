import { Plus } from "lucide-react"
import { useState } from "react"
import { isAxiosError } from "axios"

import { usersApi, type CreateUserRequest } from "@/api/users"
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
import { Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Item, ItemContent, ItemTitle, ItemDescription } from "@/components/ui/item"
import { PASSWORD_POLICY_MESSAGE, isStrongPassword } from "@/lib/password-policy"



const USER_ROLES = ["admin", "user"] as const
type UserRole = (typeof USER_ROLES)[number]
const USER_ROLE_SET = new Set<string>(USER_ROLES)

function isUserRole(value: string | null): value is UserRole {
  return value !== null && USER_ROLE_SET.has(value)
}

const UserRoleLabels: Record<UserRole, string> = {
  admin: "Admin",
  user: "User",
}

const UserRoleDescriptions: Record<UserRole, string> = {
  admin: "Full system access, can manage all users",
  user: "Regular user, access to projects assigned",
}

interface AddUserDialogProps {
  onSuccess?: () => void
}

export function AddUserDialog({ onSuccess }: AddUserDialogProps) {
  const [open, setOpen] = useState(false)
  const [isPending, setIsPending] = useState(false)
  const [formData, setFormData] = useState<CreateUserRequest>({
    username: "",
    email: "",
    password: "",
    fullname: "",
    phone: "",
    role: "user",
  })
  const [errors, setErrors] = useState<Record<string, string>>({})

  const validateForm = () => {
    const newErrors: Record<string, string> = {}

    if (!formData.username.trim()) {
      newErrors.username = "Username is required"
    }
    if (!formData.email.trim()) {
      newErrors.email = "Email is required"
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = "Invalid email format"
    }
    if (!formData.password) {
      newErrors.password = "Password is required"
    } else if (!isStrongPassword(formData.password)) {
      newErrors.password = PASSWORD_POLICY_MESSAGE
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    setIsPending(true)
    try {
      await usersApi.create(formData)
      setOpen(false)
      setFormData({
        username: "",
        email: "",
        password: "",
        fullname: "",
        phone: "",
        role: "user",
      })
      setErrors({})
      onSuccess?.()
    } catch (error) {
      const submitError = isAxiosError<{ error: string }>(error)
        ? error.response?.data?.error || error.message
        : error instanceof Error
          ? error.message
          : "Failed to create user"

      setErrors({ submit: submitError })
    } finally {
      setIsPending(false)
    }
  }

  const handleClose = () => {
    setOpen(false)
    setFormData({
      username: "",
      email: "",
      password: "",
      fullname: "",
      phone: "",
      role: "user",
    })
    setErrors({})
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>
        <Button>
          <Plus />
          Add User
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add New User</DialogTitle>
            <DialogDescription>
              Create a new user account. A default project will be created for the user.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="username">Username *</FieldLabel>
              <FieldContent>
                <Input
                  id="username"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  placeholder="Enter username"
                />
              </FieldContent>
              {errors.username && <FieldError>{errors.username}</FieldError>}
            </Field>
            <Field>
              <FieldLabel htmlFor="email">Email *</FieldLabel>
              <FieldContent>
                <Input
                  id="email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="Enter email address"
                />
              </FieldContent>
              {errors.email && <FieldError>{errors.email}</FieldError>}
            </Field>
            <Field>
              <FieldLabel htmlFor="fullname">Full Name</FieldLabel>
              <FieldContent>
                <Input
                  id="fullname"
                  value={formData.fullname}
                  onChange={(e) => setFormData({ ...formData, fullname: e.target.value })}
                  placeholder="Enter full name (optional)"
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel htmlFor="phone">Phone</FieldLabel>
              <FieldContent>
                <Input
                  id="phone"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  placeholder="Enter phone number (optional)"
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel htmlFor="password">Password *</FieldLabel>
              <FieldContent>
                <Input
                  id="password"
                  type="password"
                  autoComplete="new-password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Enter password"
                />
              </FieldContent>
              <FieldDescription>{PASSWORD_POLICY_MESSAGE}</FieldDescription>
              {errors.password && <FieldError>{errors.password}</FieldError>}
            </Field>
            <Field>
              <FieldLabel htmlFor="role">Role *</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.role}
                  onValueChange={(value) => {
                    if (isUserRole(value)) {
                      setFormData({ ...formData, role: value })
                    }
                  }}
                  itemToStringLabel={(value) => isUserRole(value) ? UserRoleLabels[value] : value ?? ""}
                >
                <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {USER_ROLES.map((role) => (
                        <ComboboxItem key={role} value={role}>
                          <Item size="xs" className="p-0">
                            <ItemContent>
                              <ItemTitle>{UserRoleLabels[role]}</ItemTitle>
                              <ItemDescription>{UserRoleDescriptions[role]}</ItemDescription>
                            </ItemContent>
                          </Item>
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            {errors.submit && (
              <FieldError className="col-span-2">{errors.submit}</FieldError>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Creating..." : "Create User"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
