<script setup lang="ts">
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { toRef, watch } from 'vue';
import * as z from 'zod';

import { updateAppImage } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import Label from '@/components/ui/label/Label.vue';
import { Separator } from '@/components/ui/separator';
import type { appModel, updateAppImageModel } from '@/types/app';
import { Info } from 'lucide-vue-next';
import { toast } from 'vue-sonner';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = toRef(props, 'app');

const profileFormSchema = toTypedSchema(z.object({
    containerImage: z
        .string()
        .min(1, {
            message: 'Container image is required.',
        }),
    registryUsername: z
        .string()
        .optional(),
    registryPassword: z
        .string()
        .optional(),
}))

const { handleSubmit, resetForm } = useForm({
    validationSchema: profileFormSchema,
    initialValues: {
        containerImage: app.value.containerImage || '',
        registryUsername: app.value.registryUsername || '',
        registryPassword: app.value.registryPassword || '',
    },
})

watch(app, async (newApp) => {
    resetForm({
        values: {
            containerImage: app.value.containerImage || '',
            registryUsername: app.value.registryUsername || '',
            registryPassword: app.value.registryPassword || '',
        }
    });
})

const onSubmit = handleSubmit(async (values) => {
    await updateAppImage(app.value.appID, values as updateAppImageModel)
    app.value.containerImage = values.containerImage
    app.value.registryUsername = values.registryUsername ?? ''
    app.value.registryPassword = values.registryPassword ?? ''
    toast.success("源码构建已更新。")
})
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            源码构建设置
        </h3>
        <p class="text-sm text-muted-foreground">
            配置构建应用程序的源码仓库地址和分支/标签信息。私有仓库需要提供相应的授权信息。
        </p>
    </div>
    <Separator />
    <form class="space-y-8" @submit="onSubmit">
        <FormField v-slot="{ componentField }" name="gitRepository">
            <FormItem>
                <FormLabel>Git 仓库地址</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="gitBranch">
            <FormItem>
                <FormLabel>Git 分支或标签</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
            </FormItem>
        </FormField>

        <Label class="text-amber-500">
            <Info class="inline mr-1 h6 w-6" />
            私有 Git 仓库需要提供以下信息：
        </Label>
        <FormField v-slot="{ componentField }" name="gitUsername">
            <FormItem>
                <FormLabel>仓库授权账号</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" autocomplete="off" />
                </FormControl>
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="gitPassword">
            <FormItem>
                <FormLabel>仓库授权密码</FormLabel>
                <FormControl>
                    <Input type="password" placeholder="" v-bind="componentField" autocomplete="no-auto-complete" />
                </FormControl>
            </FormItem>
        </FormField>

        <div class="flex gap-2 justify-start">
            <Button type="submit">
                保存
            </Button>

            <Button type="button" variant="outline" @click="resetForm">
                重置
            </Button>
        </div>
    </form>
</template>
