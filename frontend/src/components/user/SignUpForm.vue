<script setup lang="ts">
import { signUp } from "@/api/user";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type { HTMLAttributes } from "vue";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";
const router = useRouter();


const props = defineProps<{
  class?: HTMLAttributes["class"];
}>();

const errorMsg = ref('')

async function handleSubmit(event: Event) {
  event.preventDefault()
  errorMsg.value = ''
  // 获取表单数据
  const formData = new FormData(event.target as HTMLFormElement)
  const username = formData.get('username') as string
  const fullname = formData.get('fullname') as string
  const email = formData.get('email') as string
  const password = formData.get('password') as string
  const confirmPassword = formData.get('confirm-password') as string

  if (password !== confirmPassword) {
    errorMsg.value = '两次输入的密码不一致'
    return
  }
  const result = await signUp({
    username,
    fullname,
    email,
    password
  })
  if (result.success) {
    toast.dismiss()
    toast.info('注册成功！', {
      description: `${result.data.fullname || result.data.username}，现在可以登录了！`
    })
    // 注册成功后跳转到登录页面
    router.push('/sign-in')
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
              <h1 class="text-2xl font-bold">欢迎进入</h1>
              <p class="text-muted-foreground text-balance">
                注册 Ketches 账号
              </p>
            </div>
            <div class="grid gap-3">
              <Label for="username">用户名</Label>
              <Input id="username" name="username" type="text" placeholder="请输入用户名" required />
            </div>
            <div class="grid gap-3">
              <Label for="fullname">全名</Label>
              <Input id="fullname" name="fullname" type="text" placeholder="请输入全名" required />
            </div>
            <div class="grid gap-3">
              <Label for="email">邮箱</Label>
              <Input id="email" name="email" type="email" placeholder="user@example.com" required />
            </div>
            <div class="grid gap-3">
              <div class="flex items-center">
                <Label for="password">密码</Label>
              </div>
              <Input id="password" name="password" type="password" required />
            </div>
            <div class="grid gap-3">
              <div class="flex items-center">
                <Label for="confirm-password">确认密码</Label>
              </div>
              <Input id="confirm-password" name="confirm-password" type="password" required />
            </div>
            <div class="flex items-center space-x-2">
              <Checkbox id="terms" name="terms" required />
              <label for="terms" class="text-sm text-muted-foreground">
                我已阅读并同意
                <a href="#" class="underline underline-offset-4">服务条款</a> 和
                <a href="#" class="underline underline-offset-4">隐私政策</a>
              </label>
            </div>
            <Button type="submit" class="w-full"> 注册 </Button>
            <div v-if="errorMsg" class="text-red-500 text-center text-sm">{{ errorMsg }}</div>
            <div class="text-center text-sm">
              <span>已经有账户？</span>
              <RouterLink to="/sign-in" class="underline underline-offset-4">
                <span>立即登录</span>
              </RouterLink>
            </div>
          </div>
        </form>
        <div class="bg-muted relative hidden md:block">
          <img src="@/assets/placeholder.svg" alt="Image"
            class="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale" />
        </div>
      </CardContent>
    </Card>
  </div>
</template>
