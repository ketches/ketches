import { usersApi } from "@/api/users"
import { AccountAiProvidersPanel } from "@/components/account/account-ai-providers-panel"
import { PasswordForm } from "@/components/account/password-form"
import { ProfileForm } from "@/components/account/profile-form"
import { ActivitiesContent } from "@/components/activities/activities-content"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
import { Bot, Clock, Copy, Key, Pencil, UserCog, UserKey } from "lucide-react"
import * as React from "react"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { toTitleCase } from "@/lib/utils"

type AccountTab = "overview" | "security" | "ai-providers"

export function AccountPage() {
  const authUser = useAuthStore((state) => state.user)
  const updateUser = useAuthStore((state) => state.updateUser)
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = (searchParams.get("tab") as AccountTab | null) || "overview"
  const [isProfileDialogOpen, setIsProfileDialogOpen] = React.useState(false)
  const [isPasswordDialogOpen, setIsPasswordDialogOpen] = React.useState(false)

  const profileQuery = useQuery({
    queryKey: ["users", "me", authUser?.id ?? "self"],
    queryFn: usersApi.getMe,
    enabled: true,
  })

  const profileMutation = useMutation({
    mutationFn: usersApi.updateMyProfile,
    onSuccess: (updatedUser) => {
      updateUser({
        fullname: updatedUser.fullname,
        email: updatedUser.email,
        bio: updatedUser.bio,
      })
      queryClient.setQueryData(["users", "me", authUser?.id ?? "self"], updatedUser)
      void queryClient.invalidateQueries({ queryKey: ["users"] })
      toast.success("Profile updated")
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
    },
    onError: (error: AxiosError<{ error?: string }>) => {
      toast.error("Failed to update password", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const profileUser = {
    username: profileQuery.data?.username || authUser?.username || "",
    fullname: profileQuery.data?.fullname || authUser?.fullname || authUser?.username || "",
    email: profileQuery.data?.email || authUser?.email || "",
    bio: profileQuery.data?.bio || authUser?.bio || "",
    avatar: "",
  }
  const breadcrumbs: BreadcrumbItem[] = [{ label: "Account", icon: UserCog }]

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start gap-4">
          <div className="flex items-center gap-4">
            <Avatar className="h-14 w-14 rounded-lg bg-primary/10 text-primary border-none">
              <AvatarFallback className="rounded-lg text-lg font-bold">
                {(profileUser.fullname || profileUser.email || "U").charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight">{profileUser.fullname || authUser?.username || "Account"}</h1>
                {authUser?.role ? (
                  <ColorBadge color={authUser.role === "admin" ? "orange" : "blue"}>
                    {toTitleCase(authUser.role)}
                  </ColorBadge>
                ) : null}
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span>{profileUser.email}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(value) => setSearchParams({ tab: value }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview">
            <UserCog />
            Overview
          </TabsTrigger>
          <TabsTrigger value="security">
            <UserKey />
            Security
          </TabsTrigger>
          <TabsTrigger value="ai-providers">
            <Bot />
            AI Provider
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="group/card bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <UserCog className="h-4 w-4" />
                Profile
              </CardTitle>
              <CardDescription>
                Manage your account profile and contact information.
              </CardDescription>
              <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Edit profile"
                  onClick={() => setIsProfileDialogOpen(true)}
                >
                  <Pencil />
                </Button>
              </CardAction>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <p className="text-xs font-medium text-muted-foreground">Full Name</p>
                  <p className="text-sm">{profileUser.fullname || "-"}</p>
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-muted-foreground">Username</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm">{profileUser.username || "-"}</p>
                    {profileUser.username ? (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 transition-opacity group-hover/card:opacity-100"
                        onClick={() => {
                          navigator.clipboard.writeText(profileUser.username)
                          toast.success("Username copied to clipboard")
                        }}
                      >
                        <Copy />
                      </Button>
                    ) : null}
                  </div>
                </div>
                <div className="space-y-2">
                  <p className="text-xs font-medium text-muted-foreground">Email</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm">{profileUser.email || "-"}</p>
                    {profileUser.email ? (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 transition-opacity group-hover/card:opacity-100"
                        onClick={() => {
                          navigator.clipboard.writeText(profileUser.email)
                          toast.success("Email copied to clipboard")
                        }}
                      >
                        <Copy />
                      </Button>
                    ) : null}
                  </div>
                </div>
                <div className="space-y-2 sm:col-span-3">
                  <p className="text-xs font-medium text-muted-foreground">Bio</p>
                  <p className="text-sm text-muted-foreground">
                    {profileUser.bio?.trim() ? profileUser.bio : "No bio provided."}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Dialog open={isProfileDialogOpen} onOpenChange={setIsProfileDialogOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Edit Profile</DialogTitle>
                <DialogDescription>
                  Update your account profile and contact information.
                </DialogDescription>
              </DialogHeader>
              <ProfileForm
                user={profileUser}
                onSave={async (data) => {
                  await profileMutation.mutateAsync(data)
                  setIsProfileDialogOpen(false)
                }}
                isSaving={profileMutation.isPending}
                onCancel={() => setIsProfileDialogOpen(false)}
              />
            </DialogContent>
          </Dialog>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Clock className="h-4 w-4" />
                My Activities
              </CardTitle>
              <CardDescription>
                Review recent actions performed by your account.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ActivitiesContent embedded scope="personal" scopedUserId={profileQuery.data?.id} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Key className="h-4 w-4" />
                Password
              </CardTitle>
              <CardDescription>
                Keep your account secure by updating your password regularly.
              </CardDescription>
              <CardAction>
                <Button onClick={() => setIsPasswordDialogOpen(true)}>
                  <Key className="h-4 w-4" />
                  Update Password
                </Button>
              </CardAction>
            </CardHeader>
            <CardContent />
          </Card>

          <Dialog open={isPasswordDialogOpen} onOpenChange={setIsPasswordDialogOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Update Password</DialogTitle>
                <DialogDescription>
                  Enter your current password and choose a new password for your account.
                </DialogDescription>
              </DialogHeader>
              <PasswordForm
                onSave={async (data) => {
                  await passwordMutation.mutateAsync({
                    current_password: data.currentPassword,
                    new_password: data.newPassword,
                  })
                  setIsPasswordDialogOpen(false)
                }}
                isSaving={passwordMutation.isPending}
                onCancel={() => setIsPasswordDialogOpen(false)}
              />
            </DialogContent>
          </Dialog>
        </TabsContent>

        <TabsContent value="ai-providers" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Bot className="h-4 w-4" />
                AI Provider
              </CardTitle>
              <CardDescription>
                Configure personal AI providers for Builder sessions and future AI workflows.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <AccountAiProvidersPanel />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default AccountPage
