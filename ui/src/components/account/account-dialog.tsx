import { usersApi } from "@/api/users"
import { AccountAiProvidersPanel } from "@/components/account/account-ai-providers-panel"
import { AccountSettingsShell, type AccountSettingsSection } from "@/components/account/account-settings-shell"
import { PasswordForm } from "@/components/account/password-form"
import { ProfileForm } from "@/components/account/profile-form"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
import * as React from "react"
import { toast } from "sonner"

interface AccountDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: {
    fullname: string
    email: string
    bio?: string
    avatar: string
  }
}

export function AccountDialog({ open, onOpenChange, user }: AccountDialogProps) {
  const authUser = useAuthStore((state) => state.user)
  const updateUser = useAuthStore((state) => state.updateUser)
  const queryClient = useQueryClient()
  const [activeSection, setActiveSection] = React.useState<AccountSettingsSection>("profile")

  React.useEffect(() => {
    if (open) {
      setActiveSection("profile")
    }
  }, [open])

  const profileQuery = useQuery({
    queryKey: ["users", "me"],
    queryFn: usersApi.getMe,
    enabled: open && Boolean(authUser?.id),
  })

  const profileMutation = useMutation({
    mutationFn: usersApi.updateMyProfile,
    onSuccess: (updatedUser) => {
      updateUser({
        fullname: updatedUser.fullname,
        email: updatedUser.email,
        bio: updatedUser.bio,
      })
      queryClient.setQueryData(["users", "me"], updatedUser)
      void queryClient.invalidateQueries({ queryKey: ["users"] })
      toast.success("Profile updated")
      onOpenChange(false)
    },
    onError: (error: AxiosError<{ error?: string }>) => {
      toast.error("Failed to update profile", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const passwordMutation = useMutation({
    mutationFn: usersApi.updateMyPassword,
    onSuccess: () => {
      toast.success("Password updated successfully")
      onOpenChange(false)
    },
    onError: (error: AxiosError<{ error?: string }>) => {
      toast.error("Failed to update password", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const profileUser = {
    fullname: profileQuery.data?.fullname || user.fullname,
    email: profileQuery.data?.email || user.email,
    bio: profileQuery.data?.bio || user.bio || "",
    avatar: user.avatar,
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton className="sm:max-w-[50vw] w-full min-h[75vh]">
        <DialogHeader>
          <DialogTitle>Account Settings</DialogTitle>
        </DialogHeader>
        <AccountSettingsShell activeSection={activeSection} onSectionChange={setActiveSection}>
          {activeSection === "profile" ? (
            <ProfileForm
              user={profileUser}
              onSave={async (data) => {
                await profileMutation.mutateAsync(data)
              }}
              isSaving={profileMutation.isPending}
            />
          ) : null}

          {activeSection === "security" ? (
            <PasswordForm
              onSave={async (data) => {
                await passwordMutation.mutateAsync({
                  current_password: data.currentPassword,
                  new_password: data.newPassword,
                })
              }}
              isSaving={passwordMutation.isPending}
            />
          ) : null}

          {activeSection === "ai-providers" ? (
            <AccountAiProvidersPanel />
          ) : null}
        </AccountSettingsShell>
      </DialogContent>
    </Dialog>
  )
}
