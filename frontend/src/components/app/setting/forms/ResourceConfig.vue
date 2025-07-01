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

import { setAppResource } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import Label from '@/components/ui/label/Label.vue';
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { appModel } from '@/types/app';
import { Info, Save } from 'lucide-vue-next';
import { toast } from 'vue-sonner';
import { appResourceSelectOptions } from '../data/settings';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

// 保证 app 是响应式的
const app = toRef(props, 'app');

const profileFormSchema = toTypedSchema(z.object({
    replicas: z
        .number()
        .min(1, {
            message: '实例数量必须大于等于 1。',
        })
        .max(100, {
            message: '实例数量不能超过 100。',
        }),
    requestCPU: z
        .number()
        .min(0),
    requestMemory: z
        .number()
        .min(0),
    limitCPU: z
        .number()
        .min(100, {
            message: 'CPU 配额不能小于 0.1 核。',
        }),
    limitMemory: z
        .number()
        .min(128, {
            message: '内存配额不能小于 128 Mi。',
        }),
    containerCommand: z
        .string()
        .optional(),
}))

const { handleSubmit, resetForm, values } = useForm({
    validationSchema: profileFormSchema,
    initialValues: {
        replicas: app.value.replicas || 1,
        requestCPU: app.value.requestCPU,
        requestMemory: app.value.requestMemory,
        limitCPU: app.value.limitCPU || 200,
        limitMemory: app.value.limitMemory || 256,
    },
})

watch(app, (newApp) => {
    resetForm({
        values: {
            replicas: newApp.replicas || 1,
            requestCPU: newApp.requestCPU || 200,
            requestMemory: newApp.requestMemory || 256,
            limitCPU: newApp.limitCPU || 200,
            limitMemory: newApp.limitMemory || 256,
        }
    });
});

const onSubmit = handleSubmit(async (values) => {
    if (values.replicas === app.value.replicas &&
        values.requestCPU === app.value.requestCPU &&
        values.requestMemory === app.value.requestMemory &&
        values.limitCPU === app.value.limitCPU &&
        values.limitMemory === app.value.limitMemory) {
        toast.warning('没有修改任何内容。')
        return
    }

    if (values.requestCPU > values.limitCPU) {
        toast.error('最小 CPU 配额不能大于最大 CPU 配额。')
        return
    }
    if (values.requestMemory > values.limitMemory) {
        toast.error('最小内存配额不能大于最大内存配额。')
        return
    }

    const resp = await setAppResource(app.value.appID, {
        replicas: values.replicas || 1,
        requestCPU: values.requestCPU || 200,
        requestMemory: values.requestMemory || 256,
        limitCPU: values.limitCPU || 200,
        limitMemory: values.limitMemory || 256
    })

    app.value.replicas = resp.replicas
    app.value.requestCPU = resp.requestCPU
    app.value.requestMemory = resp.requestMemory
    app.value.limitCPU = resp.limitCPU
    app.value.limitMemory = resp.limitMemory
    app.value.edition = resp.edition
    toast.success('资源配置已更新。')
})
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            资源配置
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用程序的资源，包括实例数量，CPU 和内存配额等。
        </p>
    </div>
    <Separator />
    <form class="space-y-8" @submit="onSubmit">
        <FormField v-slot="{ componentField }" name="replicas">
            <FormItem>
                <FormLabel>
                    实例数量
                </FormLabel>
                <FormControl>
                    <Input type="number" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <Label class="text-blue-500">
            <Info class="inline mr-1 h6 w-6" />
            最小资源：运行应用程序运行所需最小资源
        </Label>
        <FormField v-slot="{ componentField }" name="requestCPU">
            <FormItem>
                <FormLabel>
                    最小 CPU 配额
                </FormLabel>
                <FormControl>
                    <Select v-bind="componentField">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="选择最大内存配额" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="k of appResourceSelectOptions.cpu" :key="k.value" :value="k.value">
                                    {{ k.label }}
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="requestMemory">
            <FormItem>
                <FormLabel>
                    最小内存配额
                </FormLabel>
                <FormControl>
                    <Select v-bind="componentField">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="选择最小内存配额" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="k of appResourceSelectOptions.memory" :key="k.value"
                                    :value="k.value">
                                    {{ k.label }}
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <Label class="text-blue-500">
            <Info class="inline mr-1 h6 w-6" />
            最大资源：应用程序可以使用的最大资源
        </Label>
        <FormField v-slot="{ componentField }" name="limitCPU">
            <FormItem>
                <FormLabel>
                    最大 CPU 配额
                </FormLabel>
                <FormControl>
                    <Select v-bind="componentField">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="选择最大内存配额" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="k of appResourceSelectOptions.cpu" :key="k.value" :value="k.value">
                                    {{ k.label }}
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="limitMemory">
            <FormItem>
                <FormLabel>
                    最大内存配额
                </FormLabel>
                <FormControl>
                    <Select v-bind="componentField">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="选择最大内存配额" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="k of appResourceSelectOptions.memory" :key="k.value"
                                    :value="k.value">
                                    {{ k.label }}
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
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
