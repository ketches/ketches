<script setup lang="ts">
import { signIn } from '@/api/user'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useUserStore } from '@/stores/userStore'
import type { HTMLAttributes } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'

const router = useRouter()

const props = defineProps<{
  class?: HTMLAttributes['class']
  redirectUrl?: string
}>()

const userStore = useUserStore()

async function handleSubmit(event: Event) {
  event.preventDefault()
  // 获取表单数据
  const formData = new FormData(event.target as HTMLFormElement)
  const username = formData.get('username') as string
  const password = formData.get('password') as string
  const result = await signIn(username, password)
  if (result.success) {
    if (result.data) {
      userStore.setUser(result.data)
      router.push(props.redirectUrl || { name: 'home' })
      toast.dismiss()
      toast.info('登录成功！', {
        description: `${result.data.fullname || result.data.username}，欢迎回来！`,
      })
    }
  }
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <Card class="overflow-hidden p-0">
      <CardContent class="grid p-0 md:grid-cols-2">
        <form class="p-6 md:p-8" @submit.prevent="handleSubmit">
          <div class="flex flex-col gap-6">
            <div class="flex flex-col items-center text-center">
              <h1 class="text-2xl font-bold">
                欢迎回来
              </h1>
              <p class="text-muted-foreground text-balance">
                登录 Ketches 账号
              </p>
            </div>
            <div class="grid gap-3">
              <Label for="username">用户名</Label>
              <Input id="username" name="username" type="text" class="text-sm" placeholder="请输入用户名或邮箱" required />
            </div>
            <div class="grid gap-3">
              <div class="flex items-center">
                <Label for="password">密码</Label>
                <a href="#" class="ml-auto text-sm underline-offset-2 hover:underline">
                  忘记密码？
                </a>
              </div>
              <Input id="password" name="password" type="password" class="text-sm" placeholder="请输入密码" required />
            </div>
            <Button type="submit" class="w-full">
              登录
            </Button>
            <div class="text-center text-sm">
              <span>没有账户？</span>
              <RouterLink :to="{ name: 'sign-up' }" class="underline underline-offset-4">
                <span>立即注册</span>
              </RouterLink>
            </div>
          </div>
        </form>
        <div class="bg-muted relative hidden md:block">
          <img src="@/assets/placeholder.svg" alt="Image"
            class="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale">
        </div>
      </CardContent>
    </Card>
    <div
      class="text-muted-foreground *:[a]:hover:text-primary text-center text-xs text-balance *:[a]:underline *:[a]:underline-offset-4">
      点击继续，即表示您同意我们的 <a href="#">服务条款</a> 和 <a href="#"> 隐私政策</a>。
    </div>
  </div>
</template>
