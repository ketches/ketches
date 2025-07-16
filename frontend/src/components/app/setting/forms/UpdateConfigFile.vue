<script setup lang="ts">
import { updateAppConfigFile } from '@/api/app';
import { Button } from '@/components/ui/button';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { appConfigFileModel, appModel } from '@/types/app';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { ref } from 'vue';
import { toast } from 'vue-sonner';
import * as z from 'zod';
import { configFileModeOptions } from '../data/settings';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
    configFile: {
        type: Object as () => appConfigFileModel,
        required: true,
    },
});

const emit = defineEmits(['updated', 'cancel']);

const isLoading = ref(false);

const formSchema = toTypedSchema(z.object({
    slug: z.string(),
    content: z.string().min(1, '文件内容不能为空').max(972800, '文件内容不能超过950KB'),
    mountPath: z.string().min(1, '挂载路径不能为空').max(255, '挂载路径不能超过255个字符'),
    fileMode: z.string().regex(/^0[0-7]{3}$/, '文件权限格式不正确，应为4位八进制数字，如0644'),
}));

const { handleSubmit } = useForm({
    validationSchema: formSchema,
    initialValues: {
        slug: props.configFile.slug,
        content: props.configFile.content,
        mountPath: props.configFile.mountPath,
        fileMode: props.configFile.fileMode,
    },
});

const onSubmit = handleSubmit(async (values) => {
    isLoading.value = true;
    try {
        await updateAppConfigFile(props.app.appID, props.configFile.configFileID, {
            content: values.content,
            mountPath: values.mountPath,
            fileMode: values.fileMode,
        });
        toast.success('配置文件更新成功');
        emit('updated');
    } catch (error) {
        console.error('Failed to update config file:', error);
        toast.error('更新配置文件失败');
    } finally {
        isLoading.value = false;
    }
});

function onCancel() {
    emit('cancel');
}
</script>

<template>
    <Dialog :open="true" @update:open="onCancel">
        <DialogContent class="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
            <DialogHeader>
                <DialogTitle>更新配置文件</DialogTitle>
                <DialogDescription>
                    请填写应用配置文件的相关信息。
                </DialogDescription>
            </DialogHeader>

            <form @submit="onSubmit">
                <div class="grid gap-4 py-4">
                    <FormField v-slot="{ componentField }" name="slug">
                        <FormItem>
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>配置文件标识</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>配置文件标识用于唯一标识配置文件，不能重复。</p>
                                            <li>只能包含小写字母、数字和短横线</li>
                                            <li>必须以字母开头，不能以短横线结尾</li>
                                            <li>长度为 3 到 32 个字符</li>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FormLabel>
                            <FormControl>
                                <Input placeholder="config-file" v-bind="componentField" disabled />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                    <div class="grid grid-cols-2 gap-4">
                        <FormField v-slot="{ componentField }" name="mountPath">
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>挂载路径</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>挂载路径用于指定配置文件在容器内的挂载位置。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <FormControl>
                                    <Input placeholder="/etc/config" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ componentField }" name="fileMode">
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>文件权限</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>配置文件在容器中的权限设置，默认 0644。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <Select v-bind="componentField">
                                    <FormControl>
                                        <SelectTrigger class="w-full">
                                            <SelectValue placeholder="选择文件权限" />
                                        </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                        <SelectItem v-for="option in configFileModeOptions" :key="option.value"
                                            :value="option.value">
                                            {{ option.label }}
                                        </SelectItem>
                                    </SelectContent>
                                </Select>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <FormField v-slot="{ componentField }" name="content">
                        <FormItem>
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>文件内容</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>配置文件的内容，最大支持950KB。</p>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FormLabel>
                            <FormControl>
                                <Textarea placeholder="在此输入配置文件内容..." class="min-h-32 max-h-64 font-mono text-s"
                                    v-bind="componentField" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                </div>
                <DialogFooter>
                    <Button type="button" variant="outline" @click="onCancel">
                        取消
                    </Button>
                    <Button type="submit" :disabled="isLoading">
                        {{ isLoading ? '更新中...' : '更新' }}
                    </Button>
                </DialogFooter>
            </Form>
        </DialogContent>
    </Dialog>
</template>
