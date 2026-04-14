import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { DialogFooter } from "@/components/ui/dialog"
import {
  Field,
  FieldContent,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface PasswordFormProps {
  onSave?: (data: { currentPassword: string; newPassword: string }) => Promise<void> | void
  isSaving?: boolean
  onCancel?: () => void
}

export function PasswordForm({ onSave, isSaving = false, onCancel }: PasswordFormProps) {
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()

    if (newPassword !== confirmPassword) {
      toast.error("New passwords do not match")
      return
    }

    if (newPassword.length < 8) {
      toast.error("Password must be at least 8 characters")
      return
    }

    try {
      await onSave?.({ currentPassword, newPassword })
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch {
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="currentPassword">Current Password *</FieldLabel>
          <FieldContent>
            <Input
              id="currentPassword"
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
            />
          </FieldContent>
        </Field>
        <Field>
          <FieldLabel htmlFor="newPassword">New Password *</FieldLabel>
          <FieldContent>
            <Input
              id="newPassword"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
            />
          </FieldContent>
        </Field>
        <Field>
          <FieldLabel htmlFor="confirmPassword">Confirm New Password *</FieldLabel>
          <FieldContent>
            <Input
              id="confirmPassword"
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
            />
          </FieldContent>
        </Field>
      </FieldGroup>
      <DialogFooter className="pt-4">
        {onCancel ? (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        ) : null}
        <Button type="submit" disabled={isSaving}>
          {isSaving ? "Updating..." : "Update Password"}
        </Button>
      </DialogFooter>
    </form>
  )
}
