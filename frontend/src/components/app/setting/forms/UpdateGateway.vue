<script setup lang="ts">
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle
} from '@/components/ui/dialog';
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form';
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@/components/ui/select';

import { updateAppGateway } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { appGatewayModel } from '@/types/app';
import { toTypedSchema } from '@vee-validate/zod';
import { Save } from 'lucide-vue-next';
import { useForm } from 'vee-validate';
import { computed, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
    gateway: {
        type: Object as () => appGatewayModel,
        required: true,
    },
})

const emit = defineEmits(['update:modelValue', 'gateway-updated']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const formSchema = toTypedSchema(z.object({
    port: z
        .number({
            required_error: "端口必填",
        }),
    protocol: z
        .string({
            required_error: "协议必填",
        }),
    exposed: z.boolean(),
    domain: z.string().optional(),
    path: z.string().optional(),
    gatewayPort: z
        .number({
            invalid_type_error: "网关端口必须为数字",
        })
        .min(1, "网关端口范围为 1-65535")
        .max(65535, "网关端口范围为 1-65535")
        .optional(),
}));

const { handleSubmit, resetForm, values: formValues } = useForm({
    validationSchema: formSchema,
})

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                port: props.gateway.port,
                protocol: props.gateway.protocol,
                exposed: props.gateway.exposed,
                domain: props.gateway.domain,
                path: props.gateway.path,
                gatewayPort: props.gateway.gatewayPort,
            }
        });
    }
}, { immediate: true });

const onSubmit = handleSubmit(async (values) => {
    await updateAppGateway(props.gateway.appID, props.gateway.gatewayID, {
        port: values.port,
        protocol: values.protocol,
        domain: values.domain || '',
        path: values.path || '',
        certID: props.gateway.certID,
        gatewayPort: values.gatewayPort || 0,
        exposed: values.exposed,
    });
    toast.success('网关更新成功');
    emit('gateway-updated');
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[700px]">
            <DialogHeader>
                <DialogTitle>更新应用网关</DialogTitle>
                <DialogDescription>
                    请填写应用网关的相关信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <div class="grid grid-cols-3 gap-4">
                    <FormField v-slot="{ componentField }" name="port">
                        <FormItem class="col-span-2">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>端口</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>应用容器监听的端口。</p>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FormLabel>
                            <FormControl>
                                <Input v-bind="componentField" type="number" class="w-full" disabled />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                    <FormField v-slot="{ componentField }" name="protocol">
                        <FormItem class="col-span-1">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>协议</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>端口协议：HTTP、HTTPS、TCP、UDP</p>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FormLabel>
                            <FormControl>
                                <Select v-bind="componentField">
                                    <SelectTrigger class="w-full">
                                        <SelectValue placeholder="选择协议" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectGroup>
                                            <SelectItem value="http">HTTP</SelectItem>
                                            <SelectItem value="https">HTTPS</SelectItem>
                                            <SelectItem value="tcp">TCP</SelectItem>
                                            <SelectItem value="udp">UDP</SelectItem>
                                        </SelectGroup>
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                </div>
                <FormField v-slot="{ value, handleChange }" name="exposed">
                    <FormItem class="flex flex-row items-start gap-x-3 space-y-0 rounded-md border p-4 shadow">
                        <FormControl>
                            <Checkbox :model-value="value" @update:model-value="handleChange" />
                        </FormControl>
                        <div class="space-y-1 leading-none">
                            <FormLabel>是否开启网关访问</FormLabel>
                            <FormDescription>
                                开启网关访问，将创建对外访问的网关，允许用户通过域名或端口访问应用。
                            </FormDescription>
                            <FormMessage />
                        </div>
                    </FormItem>
                </FormField>
                <div v-show="formValues.exposed" class="space-y-4">
                    <div v-show="formValues.protocol === 'http' || formValues.protocol === 'https'"
                        class="grid grid-cols-3 gap-4">
                        <FormField v-slot="{ componentField }" name="domain">
                            <FormItem class="col-span-2">
                                <FormLabel>域名</FormLabel>
                                <FormControl>
                                    <Input v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ componentField }" name="path">
                            <FormItem class="col-span-1">
                                <FormLabel>路径</FormLabel>
                                <FormControl>
                                    <Input v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div v-show="formValues.protocol === 'tcp' || formValues.protocol === 'udp'">
                        <FormField v-slot="{ componentField }" name="gatewayPort">
                            <FormItem>
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>网关端口</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>对外网关端口，允许用户通过该端口访问应用。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <FormControl>
                                    <Input v-bind="componentField" type="number" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                </div>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        <Save />
                        保存
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
