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
import { Info, Save } from 'lucide-vue-next';
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
    const resp = await updateAppImage(app.value.appID, values as updateAppImageModel)
    app.value.containerImage = resp.containerImage ?? ''
    app.value.registryUsername = resp.registryUsername ?? ''
    app.value.registryPassword = resp.registryPassword ?? ''
    app.value.updated = true
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
        <div class="grid grid-cols-3 gap-4">
            <FormField v-slot="{ componentField }" name="gitRepository">
                <FormItem class="col-span-2">
                    <FormLabel>Git 仓库地址</FormLabel>
                    <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" />
                    </FormControl>
                    <FormMessage />
                </FormItem>
            </FormField>
            <FormField v-slot="{ componentField }" name="gitBranch">
                <FormItem class="col-span-1">
                    <FormLabel>分支或标签</FormLabel>
                    <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" />
                    </FormControl>
                </FormItem>
            </FormField>
        </div>

        <Label
            class="font-normal text-amber-600 dark:text-amber-500 bg-amber-100 dark:bg-amber-950 p-2 rounded-lg mb-4 px-4">
            <Info class="inline mr-1 h4 w-4" />
            私有 Git 仓库需要提供以下信息：
        </Label>
        <div class="grid grid-cols-2 gap-4">
            <FormField v-slot="{ componentField }" name="gitUsername">
                <FormItem class="col-span-1">
                    <FormLabel>仓库授权账号</FormLabel>
                    <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" autocomplete="off" />
                    </FormControl>
                </FormItem>
            </FormField>
            <FormField v-slot="{ componentField }" name="gitPassword">
                <FormItem class="col-span-1">
                    <FormLabel>密码</FormLabel>
                    <FormControl>
                        <Input type="password" placeholder="" v-bind="componentField" autocomplete="no-auto-complete" />
                    </FormControl>
                </FormItem>
            </FormField>
        </div>

        <div class="flex gap-2 justify-start">
            <Button type="submit">
                <Save />
                保存
            </Button>
        </div>
    </form>
</template>
