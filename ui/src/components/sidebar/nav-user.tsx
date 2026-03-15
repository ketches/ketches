import { useState } from "react"

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

import { AccountDialog } from "@/components/account/account-dialog"
import { NotificationDialog } from "@/components/notifications/notification-dialog"
import { useTheme } from "@/components/theme-provider/theme-provider"
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
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
import { useAuthStore } from "@/stores/auth"
import { useNavigate } from "react-router-dom"

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
  const [accountDialogOpen, setAccountDialogOpen] = useState(false)
  const { unreadCount, dialogOpen: notifDialogOpen, setDialogOpen: setNotifDialogOpen } = useNotifications()

  const authUser = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()

  const user = {
    name: authUser?.fullname || authUser?.username || initialUser.name,
    email: authUser?.email || initialUser.email,
    avatar: initialUser.avatar,
  }

  const handleLogout = () => {
    markManualLogout()
    logout()
    navigate("/login", { replace: true })
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
                  <Avatar className="h-8 w-8 rounded-lg">
                    <AvatarImage src={user.avatar} alt={user.name} />
                    <AvatarFallback className="rounded-lg">{user.name.charAt(0).toUpperCase()}</AvatarFallback>
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
                    <Avatar className="h-8 w-8 rounded-lg">
                      <AvatarImage src={user.avatar} alt={user.name} />
                      <AvatarFallback className="rounded-lg">{user.name.charAt(0).toUpperCase()}</AvatarFallback>
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
                <DropdownMenuItem onClick={() => setAccountDialogOpen(true)}>
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
                <DropdownMenuItem onClick={handleLogout}>
                  <LogOut className="mr-2" />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      <AccountDialog
        open={accountDialogOpen}
        onOpenChange={setAccountDialogOpen}
        user={user}
      />
      <NotificationDialog
        open={notifDialogOpen}
        onOpenChange={setNotifDialogOpen}
      />
    </>
  )
}
