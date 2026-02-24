import { Key } from "lucide-react"
import { useState } from "react"

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

interface EditPasswordDialogProps {
  userId: string
  username: string
  onSubmit: (password: string) => void
  isPending?: boolean
}

export function EditPasswordDialog({ username, onSubmit, isPending }: EditPasswordDialogProps) {
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (password && password === confirmPassword) {
      onSubmit(password)
      setPassword("")
      setConfirmPassword("")
      setOpen(false)
    }
  }

  const isValid = password.length >= 6 && password === confirmPassword

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>
        <Button
          variant="ghost"
          size="icon-sm"
          className="text-muted-foreground hover:text-foreground"
        >
          <Key />
          <span className="sr-only">Edit password</span>
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Change Password</DialogTitle>
            <DialogDescription>
              Set a new password for user <span className="font-medium text-foreground">{username}</span>
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="password">New Password *</FieldLabel>
              <FieldContent>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter new password (min 6 characters)"
                  autoComplete="new-password"
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel htmlFor="confirm-password">Confirm Password *</FieldLabel>
              <FieldContent>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Confirm new password"
                  autoComplete="new-password"
                />
              </FieldContent>
              {password && confirmPassword && password !== confirmPassword && (
                <FieldError>Passwords do not match</FieldError>
              )}
            </Field>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setOpen(false)
                setPassword("")
                setConfirmPassword("")
              }}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!isValid || isPending}>
              {isPending ? "Updating..." : "Update Password"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
