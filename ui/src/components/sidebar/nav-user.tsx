import { authApi } from "@/api/auth"
import {
  Bell,
  ChevronsUpDown,
  Languages,
  LogOut,
  Monitor,
  Moon,
  PaletteIcon,
  Sun,
  UserCog
} from "lucide-react"

import { NotificationDialog } from "@/components/notifications/notification-dialog"
import { useTheme } from "@/components/theme-provider/theme-provider"
import {
  Avatar,
  AvatarImage
} from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { useNotifications } from "@/hooks/use-notifications"
import { markManualLogout } from "@/lib/auth-redirect"
import { getErrorMessage } from "@/lib/utils"
import { logoutSession } from "@/lib/logout-session"
import { useAuthStore } from "@/stores/auth"
import { useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { UserAvatarFallback } from "../shared/user-avatar"

export function NavUser({
  user: initialUser,
}: {
  user: {
    name: string
    email: string
    avatar: string
  }
}) {
  const { isMobile } = useSidebar()
  const { setTheme } = useTheme()
  const theme = useTheme().theme
  const { unreadCount, dialogOpen: notifDialogOpen, setDialogOpen: setNotifDialogOpen } = useNotifications()

  const authUser = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [isLoggingOut, setIsLoggingOut] = React.useState(false)

  const user = {
    name: authUser?.fullname || authUser?.username || initialUser.name,
    email: authUser?.email || initialUser.email,
    avatar: initialUser.avatar,
  }

  const handleLogout = async () => {
    if (isLoggingOut) return
    setIsLoggingOut(true)
    try {
      await logoutSession({
        requestLogout: authApi.logout,
        markManualLogout,
        clearQueries: () => queryClient.clear(),
        clearAuth: logout,
        navigateToLogin: () => navigate("/login", { replace: true }),
      })
    } catch (error: unknown) {
      toast.error("Failed to log out", {
        description: getErrorMessage(error, "Please try again."),
      })
    } finally {
      setIsLoggingOut(false)
    }
  }

  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <SidebarMenuButton
                  size="lg"
                  className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                >
                  <Avatar className="h-8 w-8 after:border-none">
                    {user.avatar && <AvatarImage src={user.avatar} alt={user.name} />}
                    <UserAvatarFallback name={user.name} className="rounded-full text-xs font-bold" />
                    {/* <AvatarFallback>{user.name?.charAt(0).toUpperCase() || "U"}</AvatarFallback> */}
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user.name}</span>
                    <span className="truncate text-xs">{user.email}</span>
                  </div>
                  <ChevronsUpDown className="ml-auto size-4" />
                </SidebarMenuButton>
              }
            />
            <DropdownMenuContent
              className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
              side={isMobile ? "bottom" : "right"}
              align="end"
              sideOffset={4}
            >
              <DropdownMenuGroup>
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar className="h-8 w-8 after:border-none">
                      {user.avatar && <AvatarImage src={user.avatar} alt={user.name} />}
                      <UserAvatarFallback name={user.name} className="rounded-full text-xs font-bold" />
                      {/* <AvatarFallback>{user.name?.charAt(0).toUpperCase() || "U"}</AvatarFallback> */}
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">{user.name}</span>
                      <span className="truncate text-xs">{user.email}</span>
                    </div>
                  </div>
                </DropdownMenuLabel>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => navigate("/account")}>
                  <UserCog className="mr-2" />
                  Account
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setNotifDialogOpen(true)}>
                  <Bell className="mr-2" />
                  Notifications
                  {unreadCount > 0 && (
                    <span className="ml-auto flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[0.5rem] font-medium text-white hover:text-white">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <PaletteIcon className="mr-2" />
                    <span>Appearance</span>
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuGroup>
                      <DropdownMenuRadioGroup value={theme}
                        onValueChange={setTheme}>
                        <DropdownMenuRadioItem value="system">
                          <Monitor className="mr-2" />
                          <span>System</span>
                        </DropdownMenuRadioItem>
                        <DropdownMenuRadioItem value="light">
                          <Sun className="mr-2" />
                          <span>Light</span>
                        </DropdownMenuRadioItem>
                        <DropdownMenuRadioItem value="dark">
                          <Moon className="mr-2" />
                          <span>Dark</span>
                        </DropdownMenuRadioItem>
                      </DropdownMenuRadioGroup>
                    </DropdownMenuGroup>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <Languages className="mr-2" />
                    <span>Language</span>
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuGroup>
                      <DropdownMenuRadioGroup value="en">
                        <DropdownMenuRadioItem value="en">
                          <span>English</span>
                        </DropdownMenuRadioItem>
                        <DropdownMenuRadioItem value="zh-Hans">
                          <span>简体中文</span>
                        </DropdownMenuRadioItem>
                        <DropdownMenuRadioItem value="zh-Hant">
                          <span>繁體中文</span>
                        </DropdownMenuRadioItem>
                      </DropdownMenuRadioGroup>
                    </DropdownMenuGroup>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => void handleLogout()} disabled={isLoggingOut}>
                  <LogOut className="mr-2" />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      <NotificationDialog
        open={notifDialogOpen}
        onOpenChange={setNotifDialogOpen}
      />
    </>
  )
}
