import { Plus } from "lucide-react"
import { useState } from "react"

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
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const USER_ROLES = ["admin", "user"] as const
type UserRole = (typeof USER_ROLES)[number]

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
    } else if (formData.password.length < 6) {
      newErrors.password = "Password must be at least 6 characters"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
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
    } catch (error: any) {
      setErrors({ submit: error.response?.data?.error || "Failed to create user" })
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
                  placeholder="Enter password (min 6 characters)"
                />
              </FieldContent>
              {errors.password && <FieldError>{errors.password}</FieldError>}
            </Field>
            <Field>
              <FieldLabel htmlFor="role">Role *</FieldLabel>
              <FieldContent>
                <Select
                  value={formData.role}
                  onValueChange={(value) => value && setFormData({ ...formData, role: value })}
                >
                  <SelectTrigger id="role" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {USER_ROLES.map((role) => (
                      <SelectItem key={role} value={role}>
                        <div className="flex flex-col">
                          <span>{UserRoleLabels[role]}</span>
                          <span className="text-xs text-muted-foreground">
                            {UserRoleDescriptions[role]}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
