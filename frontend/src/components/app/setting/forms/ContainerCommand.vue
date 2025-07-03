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
import * as z from 'zod';

import { setAppCommand } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import type { appModel } from '@/types/app';
import { Info, Save } from 'lucide-vue-next';
import { toRef, watch } from 'vue';
import { toast } from 'vue-sonner';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = toRef(props, 'app');

const profileFormSchema = toTypedSchema(z.object({
    containerCommand: z
        .string()
        .optional(),
}))

const { handleSubmit, resetForm } = useForm({
    validationSchema: profileFormSchema,
    initialValues: {
        containerCommand: app.value.containerCommand || '',
    },
})

watch(app, (newApp) => {
    resetForm({
        values: {
            containerCommand: app.value.containerCommand || '',
        }
    });
});

const onSubmit = handleSubmit(async (values) => {
    if (values.containerCommand === app.value.containerCommand) {
        toast.warning('没有修改任何内容。')
        return
    }
    const resp = await setAppCommand(app.value.appID, { containerCommand: values.containerCommand ?? '' })
    app.value.containerCommand = resp.containerCommand ?? ''
    app.value.updated = true
    toast.success('启动命令已更新。')
})
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            容器启动命令
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用程序的容器启动命令。
        </p>
    </div>
    <Separator />
    <form class="space-y-8" @submit="onSubmit">
        <FormField v-slot="{ componentField }" name="containerCommand">
            <FormItem>
                <FormLabel>
                    启动命令
                    <label class="flex items-center text-sm font-normal text-amber-500">
                        <Info class="inline mr-1 h-4 w-4" />
                        <span>设置此项会覆盖容器镜像默认的启动命令。</span>
                    </label>
                </FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <div class="flex gap-2 justify-start">
            <Button type="submit">
                <Save />
                保存
            </Button>
        </div>
    </form>
</template>
