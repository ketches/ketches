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

import { createAppGateway } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { toTypedSchema } from '@vee-validate/zod';
import { Plus } from 'lucide-vue-next';
import { useForm } from 'vee-validate';
import { computed, watch } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
    appID: {
        type: String,
        required: true,
    },
})

const emit = defineEmits(['update:modelValue', 'gateway-created']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const formSchema = toTypedSchema(z.object({
    port: z.number().min(1).max(65535),
    protocol: z.string(),
    exposed: z.boolean(),
    domain: z.string().optional(),
    path: z.string().optional(),
    gatewayPort: z.number().min(1).max(65535).optional(),
}));

const { isFieldDirty, handleSubmit, resetForm, values: formValues } = useForm({
    validationSchema: formSchema,
})

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                port: 80,
                protocol: 'HTTP',
                exposed: false,
                domain: '',
                path: '/',
            }
        });
    }
})

const onSubmit = handleSubmit(async (values) => {
    await createAppGateway(props.appID, {
        port: values.port,
        protocol: values.protocol,
        domain: values.domain || '',
        path: values.path || '',
        certID: '',
        gatewayPort: values.gatewayPort || 0,
        exposed: values.exposed,
    });
    toast.success('网关创建成功');
    emit('gateway-created');
    open.value = false;
})
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[600px]">
            <DialogHeader>
                <DialogTitle>创建应用网关</DialogTitle>
                <DialogDescription>
                    请填写应用网关的相关信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <div class="grid grid-cols-3 gap-4">
                    <FormField v-slot="{ componentField }" name="port" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-2">
                            <FormLabel>端口</FormLabel>
                            <FormControl>
                                <Input v-bind="componentField" type="number" class="w-full" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                    <FormField v-slot="{ componentField }" name="protocol" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-1">
                            <FormLabel>协议</FormLabel>
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
                <div v-if="formValues.exposed">
                    <div v-if="formValues.protocol === 'http' || formValues.protocol === 'https'"
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
                    <FormField v-if="formValues.protocol === 'tcp' || formValues.protocol === 'udp'"
                        v-slot="{ componentField }" name="gatewayPort">
                        <FormItem>
                            <FormLabel>网关端口</FormLabel>
                            <FormControl>
                                <Input v-bind="componentField" type="number" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                </div>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        <Plus />
                        创建
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
