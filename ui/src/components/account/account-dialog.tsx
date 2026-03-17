import { usersApi } from "@/api/users"
import { PasswordForm } from "@/components/account/password-form"
import { ProfileForm } from "@/components/account/profile-form"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
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
      <DialogContent showCloseButton>
        <DialogHeader>
          <DialogTitle>Account Settings</DialogTitle>
        </DialogHeader>
        <Tabs defaultValue="profile">
          <TabsList className="grid w-full grid-cols-2 gap-2">
            <TabsTrigger value="profile">Profile</TabsTrigger>
            <TabsTrigger value="password">Password</TabsTrigger>
          </TabsList>
          <TabsContent value="profile">
            <ProfileForm
              user={profileUser}
              onSave={async (data) => {
                await profileMutation.mutateAsync(data)
              }}
              isSaving={profileMutation.isPending}
            />
          </TabsContent>
          <TabsContent value="password">
            <PasswordForm
              onSave={async (data) => {
                await passwordMutation.mutateAsync({
                  current_password: data.currentPassword,
                  new_password: data.newPassword,
                })
              }}
              isSaving={passwordMutation.isPending}
            />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}
