import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { DialogFooter } from "@/components/ui/dialog"
import {
  Field,
  FieldContent,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"

interface ProfileFormProps {
  user: {
    fullname: string
    email: string
    bio?: string
    avatar: string
  }
  onSave?: (data: { fullname: string; email: string; bio: string }) => Promise<void> | void
  isSaving?: boolean
  onCancel?: () => void
}

export function ProfileForm({ user, onSave, isSaving = false, onCancel }: ProfileFormProps) {
  const [fullname, setFullname] = useState(user.fullname)
  const [email, setEmail] = useState(user.email)
  const [bio, setBio] = useState(user.bio ?? "")

  useEffect(() => {
    setFullname(user.fullname)
    setEmail(user.email)
    setBio(user.bio ?? "")
  }, [user.bio, user.email, user.fullname])

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    await onSave?.({ fullname, email, bio })
  }

  return (
    <form onSubmit={handleSubmit}>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="fullname">Full Name *</FieldLabel>
          <FieldContent>
            <Input
              id="fullname"
              value={fullname}
              onChange={(e) => setFullname(e.target.value)}
              placeholder="Your full name"
              required
            />
          </FieldContent>
        </Field>
        <Field>
          <FieldLabel htmlFor="email">Email *</FieldLabel>
          <FieldContent>
            <Input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="your@email.com"
              required
            />
          </FieldContent>
        </Field>
        <Field>
          <FieldLabel htmlFor="bio">Bio</FieldLabel>
          <FieldContent>
            <Textarea
              id="bio"
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Tell us about yourself"
              rows={4}
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
          {isSaving ? "Saving..." : "Save Changes"}
        </Button>
      </DialogFooter>
    </form >
  )
}
