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
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';

import { createAppVolume } from '@/api/app';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { toTypedSchema } from '@vee-validate/zod';
import { HardDrive, SquareDashed, SquareDot, SquaresIntersect, SquaresSubtract } from 'lucide-vue-next';
import { useForm } from 'vee-validate';
import { computed, ref, watch } from 'vue';
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

const emit = defineEmits(['update:modelValue', 'volume-created']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const formSchema = toTypedSchema(z.object({
    slug: z
        .string()
        .min(1, {
            message: '存储标识是必填项。',
        }).max(32, {
            message: '存储标识不能超过32个字符。',
        })
        .max(32),
    mountPath: z
        .string()
        .min(1, {
            message: '挂载路径是必填项。',
        }),
    subPath: z.string().optional(),
    volumeType: z
        .string().optional(),
    storageClass: z.string().optional(),
    capacity: z
        .number()
        .default(1),
    accessModes: z
        .array(z.string()).min(1, {
            message: '至少选择一个访问模式。',
        }),
    volumeMode: z
        .string()
        .default('Filesystem'),
}));

const { isFieldDirty, handleSubmit, resetForm } = useForm({
    validationSchema: formSchema,
})

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                slug: '',
                mountPath: '',
                subPath: '',
                volumeType: 'PersistentVolumeClaim',
                storageClass: '',
                capacity: 1,
                accessModes: ['ReadWriteOnce'],
                volumeMode: 'Filesystem',
            }
        });
    }
})

const onSubmit = handleSubmit(async (values) => {
    await createAppVolume(props.appID, {
        slug: values.slug,
        mountPath: values.mountPath,
        subPath: values.subPath,
        volumeType: values.volumeType,
        storageClass: values.storageClass,
        capacity: values.capacity,
        accessModes: values.accessModes,
        volumeMode: values.volumeMode,
    });
    toast.success('存储卷创建成功！');
    open.value = false;
})

const subPath = ref(false);
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[500px]">
            <DialogHeader>
                <DialogTitle>创建应用存储卷</DialogTitle>
                <DialogDescription>
                    请填写应用存储卷的相关信息。
                </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储卷标识
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>存储卷标识用于唯一标识存储卷，不能重复。</p>
                                        <li>只能包含小写字母、数字和短横线</li>
                                        <li>必须以字母开头</li>
                                        <li>不能以短横线结尾。</li>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="mountPath" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        挂载路径
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>挂载路径用于指定存储卷在容器内的挂载位置。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <p class="text-sm text-muted-foreground flex items-center gap-2">
                    <Checkbox v-model="subPath" />
                    <label class="">
                        需要挂载子路径
                    </label>
                </p>
                <FormField v-if="subPath" v-slot="{ componentField }" name="subPath" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储到子路径
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>用于指定容器挂载存储卷的子路径。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="volumeType" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储卷类型
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>选择存储卷类型，默认为 PersistentVolumeClaim。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Select v-bind="componentField" :default-value="'persistentVolumeClaim'">
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择存储卷类型" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem value="persistentVolumeClaim">
                                            <HardDrive />
                                            持久化存储(默认)
                                        </SelectItem>
                                        <SelectItem value="emptyDir">
                                            <SquaresIntersect />
                                            临时存储(多容器共享，重启数据将丢失)
                                        </SelectItem>
                                        <SelectItem value="local" disabled>
                                            <SquaresSubtract />
                                            本地存储(实例漂移后数据可能丢失)
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="storageClass" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储类
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>指定存储卷的存储类，默认为空。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" class="w-full" placeholder="例如：standard" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="capacity" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储容量
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>指定存储卷的容量，单位为 GiB，默认为 1 GiB。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Input v-bind="componentField" type="number" :default-value="1" class="w-full" />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="accessModes" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        访问模式
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>选择存储卷的访问模式，至少选择一个。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Select v-bind="componentField" multiple>
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择访问模式" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem value="ReadWriteOnce">
                                            <SquareDashed />
                                            单节点访问
                                        </SelectItem>
                                        <SelectItem value="ReadOnlyMany">
                                            <SquareDot />
                                            多节点访问
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="volumeMode" :validate-on-blur="!isFieldDirty">
                    <FormItem>
                        <FormLabel>
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger class="hover:bg-secondary">
                                        存储模式
                                    </TooltipTrigger>
                                    <TooltipContent side="right">
                                        <p>选择存储卷的模式，默认为 Filesystem。</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        </FormLabel>
                        <FormControl>
                            <Select v-bind="componentField" :default-value="'Filesystem'">
                                <SelectTrigger class="w-full">
                                    <SelectValue placeholder="选择卷模式" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem value="Filesystem">
                                            <SquareDashed />
                                            文件系统
                                        </SelectItem>
                                        <SelectItem value="Block">
                                            <SquareDot />
                                            块存储
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                </FormField>
                <DialogFooter>
                    <Button type="submit" class="w-full">
                        创建
                    </Button>
                </DialogFooter>
            </form>
        </DialogContent>
    </Dialog>
</template>
