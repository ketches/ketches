import { useState } from "react"

import { Button } from "@/components/ui/button"
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
}

export function PasswordForm({ onSave, isSaving = false }: PasswordFormProps) {
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [error, setError] = useState("")

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")

    if (newPassword !== confirmPassword) {
      setError("New passwords do not match")
      return
    }

    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters")
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
        {error && (
          <div className="text-destructive text-sm">{error}</div>
        )}
        <Field>
          <Button type="submit" disabled={isSaving}>
            {isSaving ? "Updating..." : "Update Password"}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  )
}
