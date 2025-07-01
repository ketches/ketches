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
            message: '容器镜像地址是必填项。',
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

watch(app, (newApp) => {
    resetForm({
        values: {
            containerImage: app.value.containerImage || '',
            registryUsername: app.value.registryUsername || '',
            registryPassword: app.value.registryPassword || '',
        }
    });
});

const onSubmit = handleSubmit(async (values) => {
    const resp = await updateAppImage(app.value.appID, values as updateAppImageModel)
    app.value.containerImage = resp.containerImage
    app.value.registryUsername = resp.registryUsername ?? ''
    app.value.registryPassword = resp.registryPassword ?? ''
    app.value.edition = resp.edition ?? ''
    toast.success('容器镜像已更新。')
})
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            容器镜像设置
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用程序的容器镜像地址，私有镜像需提供相应的授权信息。
        </p>
    </div>
    <Separator />
    <form class="space-y-8" @submit="onSubmit">
        <FormField v-slot="{ componentField }" name="containerImage">
            <FormItem>
                <FormLabel>镜像地址</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <Label
            class="font-normal text-amber-600 dark:text-amber-500 bg-amber-100 dark:bg-amber-950 p-2 rounded-lg mb-4 px-4">
            <Info class="inline mr-1 h4 w-4" />
            私有镜像仓库需要提供以下信息：
        </Label>
        <div class="grid grid-cols-2 gap-4">
            <FormField v-slot="{ componentField }" name="registryUsername">
                <FormItem class="col-span-1">
                    <FormLabel>镜像仓库授权账号</FormLabel>
                    <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" autocomplete="off" />
                    </FormControl>
                </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="registryPassword">
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
