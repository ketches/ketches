<script setup lang="ts">
import { signOut } from '@/api/user'
import {
  Avatar,
  AvatarFallback
} from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useUserStore } from '@/stores/userStore'
import { useColorMode } from '@vueuse/core'
import {
  BadgeCheck,
  Bell,
  Bug,
  Check,
  ChevronsUpDown,
  Github,
  LogOut,
  MonitorCog,
  Moon,
  Palette,
  Sun
} from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'

const router = useRouter()

const { isMobile } = useSidebar()

const mode = useColorMode()

const appearance = localStorage.getItem('vueuse-color-scheme') || 'auto'

const { user } = storeToRefs(useUserStore())

function handleSignOut() {
  signOut().then(() => {
    toast.dismiss()
    toast.info('成功退出', {
      description: `${user.value?.fullname || user.value?.username}，期待下次再见！`,
    })
    router.push({ name: 'sign-in', query: { redirect: router.currentRoute.value.fullPath } })
  })
}
</script>

<template>
  <SidebarMenu>
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <SidebarMenuButton size="lg"
            class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
            <Avatar class="h-8 w-8 rounded-lg">
              <!-- <AvatarImage :src="user.avatar" :alt="user.username" /> -->
              <AvatarFallback class="rounded-lg">
                {{ (user && (user.fullname?.charAt(0)?.toUpperCase() || user.username?.charAt(0)?.toUpperCase())) || ''
                }}
              </AvatarFallback>
            </Avatar>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-medium">{{ user?.username }}</span>
              <span class="truncate text-xs">{{ user?.email }}</span>
            </div>
            <ChevronsUpDown class="ml-auto size-4" />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent class="w-[--reka-dropdown-menu-trigger-width] min-w-56 rounded-lg"
          :side="isMobile ? 'bottom' : 'right'" align="end" :side-offset="4">
          <DropdownMenuLabel class="p-0 font-normal">
            <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
              <Avatar class="h-8 w-8 rounded-lg">
                <AvatarFallback class="rounded-lg">
                  {{ (user && (user.fullname?.charAt(0)?.toUpperCase() || user.username?.charAt(0)?.toUpperCase())) ||
                    '' }}
                </AvatarFallback>
              </Avatar>
              <div class="grid flex-1 text-left text-sm leading-tight">
                <span class="truncate font-semibold">{{ user?.username }}</span>
                <span class="truncate text-xs">{{ user?.email }}</span>
              </div>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem as="a" href="https://github.com/ketches/ketches" target="_blank"
              rel="noopener noreferrer">
              <Github />
              Ketches 开源
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuGroup>
            <DropdownMenuItem as="a" href="https://github.com/ketches/ketches/issues/new" target="_blank"
              rel="noopener noreferrer">
              <Bug />
              报告问题
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem>
              <BadgeCheck />
              账号设置
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Bell />
              消息
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              <Palette class="mr-2 h-4 w-4 text-muted-foreground" />
              <span>外观</span>
            </DropdownMenuSubTrigger>
            <DropdownMenuPortal>
              <DropdownMenuSubContent>
                <DropdownMenuItem :disabled="appearance === 'light'" @click="mode = 'light'; appearance = 'light'">
                  <Sun />
                  <span>浅色模式</span>
                  <Check v-if="appearance === 'light'" class="ml-auto h-4 w-4" />
                </DropdownMenuItem>
                <DropdownMenuItem :disabled="appearance === 'dark'" @click="mode = 'dark'; appearance = 'dark'">
                  <Moon />
                  <span>深色模式</span>
                  <Check v-if="appearance === 'dark'" class="ml-auto h-4 w-4" />
                </DropdownMenuItem>
                <DropdownMenuItem :disabled="appearance === 'auto'" @click="mode = 'auto'; appearance = 'auto'">
                  <MonitorCog />
                  <span>跟随系统</span>
                  <Check v-if="appearance === 'auto'" class="ml-auto h-4 w-4" />
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuPortal>
          </DropdownMenuSub>
          <DropdownMenuItem @click="handleSignOut">
            <LogOut />
            退出登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
