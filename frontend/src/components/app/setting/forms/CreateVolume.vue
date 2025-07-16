<script setup lang="ts">
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectItemText,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";

import { createAppVolume } from "@/api/app";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toTypedSchema } from "@vee-validate/zod";
import { Plus } from "lucide-vue-next";
import { useForm } from "vee-validate";
import { computed, watch } from "vue";
import { toast } from "vue-sonner";
import * as z from "zod";
import {
    accessModeRefs,
    volumeModeRefs,
    volumeTypeRefs,
} from "../data/settings";

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
    appID: {
        type: String,
        required: true,
    },
});

const emit = defineEmits(["update:modelValue", "volume-created"]);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit("update:modelValue", value);
    },
});

const formSchema = toTypedSchema(
    z.object({
        slug: z
            .string({
                required_error: "存储卷标识必填",
            })
            .min(3, "长度不能小于 3")
            .max(32, "长度不能大于 32")
            .regex(/^[a-z]/, "必须以小写字母开头")
            .regex(/[a-z0-9]$/, "不能以短横线结尾")
            .regex(/^[a-z0-9-]+$/, "只能包含小写字母、数字和短横线"),
        mountPath: z.string().min(1, {
            message: "挂载路径是必填项。",
        }),
        subPath: z.string().optional(),
        volumeType: z.string().optional(),
        storageClass: z.string().optional(),
        capacity: z.number().default(1),
        accessModes: z.array(z.string()).min(1, {
            message: "至少选择一个访问模式。",
        }),
        volumeMode: z.string().default("Filesystem"),
    })
);

const { isFieldDirty, handleSubmit, resetForm, values: formValues, setFieldValue } = useForm({
    validationSchema: formSchema,
});

watch(open, (isOpen) => {
    if (isOpen) {
        resetForm({
            values: {
                slug: "",
                mountPath: "",
                subPath: "",
                volumeType: "pvc",
                storageClass: "",
                capacity: 1,
                accessModes: ["ReadWriteOnce"],
                volumeMode: "Filesystem",
            },
        });
    }
});

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
    toast.success("存储卷创建成功！");
    emit("volume-created");
    open.value = false;
});
</script>

<template>
    <Dialog :open="open" @update:open="open = $event">
        <DialogContent class="sm:max-w-[700px]">
            <DialogHeader>
                <DialogTitle>创建应用存储卷</DialogTitle>
                <DialogDescription> 请填写应用存储卷的相关信息。 </DialogDescription>
            </DialogHeader>
            <form class="space-y-6" @submit="onSubmit">
                <div class="grid grid-cols-2 gap-4">
                    <FormField v-slot="{ componentField }" name="slug" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-1">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>存储卷标识</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>存储卷标识用于唯一标识存储卷，不能重复。</p>
                                            <li>只能包含小写字母、数字和短横线</li>
                                            <li>必须以字母开头，不能以短横线结尾</li>
                                            <li>长度为 3 到 32 个字符</li>
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
                    <FormField v-slot="{ value, handleChange }" name="volumeType" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-1">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>存储卷类型</TooltipTrigger>
                                        <TooltipContent side="right">
                                            <p>选择存储卷类型，默认为 PersistentVolumeClaim。</p>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FormLabel>
                            <FormControl>
                                <Select :model-value="value" @update:model-value="handleChange">
                                    <FormControl>
                                        <SelectTrigger class="w-full">
                                            <SelectValue>
                                                <div v-if="value" class="flex items-center">
                                                    <component :is="volumeTypeRefs[
                                                        value as keyof typeof volumeTypeRefs
                                                    ]?.icon" class="h-4 w-4 mr-2" />
                                                    <span>{{ volumeTypeRefs[value as keyof typeof volumeTypeRefs]?.label
                                                        }}</span>
                                                </div>
                                                <span v-else>选择存储卷类型</span>
                                            </SelectValue>
                                        </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                        <SelectGroup>
                                            <SelectItem v-for="(volumeType, key) in volumeTypeRefs" :key="key"
                                                :value="key">
                                                <div class="flex items-center gap-3">
                                                    <component :is="volumeType.icon" class="h-4 w-4" />
                                                    <div class="flex flex-col">
                                                        <SelectItemText>{{
                                                            volumeType.label
                                                            }}</SelectItemText>
                                                        <span class="text-xs text-muted-foreground">{{
                                                            volumeType.desc
                                                            }}</span>
                                                    </div>
                                                </div>
                                            </SelectItem>
                                        </SelectGroup>
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                </div>
                <div class="grid grid-cols-3 gap-4">
                    <FormField v-slot="{ componentField }" name="mountPath" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-2">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>挂载路径</TooltipTrigger>
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
                    <FormField v-slot="{ componentField }" name="subPath" :validate-on-blur="!isFieldDirty">
                        <FormItem class="col-span-1">
                            <FormLabel>
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger>子路径（可选）</TooltipTrigger>
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
                </div>
                <div v-show="formValues.volumeType === 'pvc'" class="space-y-6">
                    <div class="grid grid-cols-2 gap-4">
                        <FormField v-slot="{ componentField }" name="storageClass" :validate-on-blur="!isFieldDirty">
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>存储类</TooltipTrigger>
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
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>存储容量</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>指定存储卷的容量，单位为 GiB，默认为 1 GiB。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <FormControl>
                                    <Input v-bind="componentField" type="number" class="w-full" default-value="1" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                        <FormField v-slot="{ componentField }" name="accessModes" :validate-on-blur="!isFieldDirty">
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>访问模式</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>选择存储卷的访问模式，至少选择一个。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <FormControl>
                                    <Select v-bind="componentField" multiple :default-value="['ReadWriteOnce']">
                                        <FormControl>
                                            <SelectTrigger class="w-full">
                                                <SelectValue>
                                                    <div v-if="componentField.modelValue" class="flex items-center">
                                                        <component :is="accessModeRefs[
                                                            componentField.modelValue as keyof typeof accessModeRefs
                                                        ]?.icon" class="h-4 w-4 mr-2" />
                                                        <span>{{
                                                            accessModeRefs[
                                                                componentField.modelValue as keyof typeof accessModeRefs
                                                            ]?.label
                                                        }}</span>
                                                    </div>
                                                    <span v-else>选择访问模式</span>
                                                </SelectValue>
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectGroup>
                                                <SelectItem v-for="(accessMode, key) in accessModeRefs" :key="key"
                                                    :value="key">
                                                    <div class="flex items-center gap-3">
                                                        <component :is="accessMode.icon" class="h-4 w-4" />
                                                        <div class="flex flex-col">
                                                            <SelectItemText>{{
                                                                accessMode.label
                                                                }}</SelectItemText>
                                                            <span class="text-xs text-muted-foreground">{{
                                                                accessMode.desc
                                                                }}</span>
                                                        </div>
                                                    </div>
                                                </SelectItem>
                                            </SelectGroup>
                                        </SelectContent>
                                    </Select>
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ value, handleChange }" name="volumeMode" :validate-on-blur="!isFieldDirty">
                            <FormItem class="col-span-1">
                                <FormLabel>
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger>存储模式</TooltipTrigger>
                                            <TooltipContent side="right">
                                                <p>选择存储卷的模式，默认为 Filesystem。</p>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                </FormLabel>
                                <FormControl>
                                    <Select :model-value="value" @update:model-value="handleChange"
                                        :default-value="'Filesystem'">
                                        <FormControl>
                                            <SelectTrigger class="w-full">
                                                <SelectValue>
                                                    <div v-if="value" class="flex items-center">
                                                        <component :is="volumeModeRefs[
                                                            value as keyof typeof volumeModeRefs
                                                        ]?.icon
                                                            " class="h-4 w-4 mr-2" />
                                                        <span>{{
                                                            volumeModeRefs[
                                                                value as keyof typeof volumeModeRefs
                                                            ]?.label
                                                        }}</span>
                                                    </div>
                                                    <span v-else>选择访问模式</span>
                                                </SelectValue>
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectGroup>
                                                <SelectItem v-for="(volumeMode, key) in volumeModeRefs" :key="key"
                                                    :value="key">
                                                    <div class="flex items-center gap-3">
                                                        <component :is="volumeMode.icon" class="h-4 w-4" />
                                                        <div class="flex flex-col">
                                                            <SelectItemText>{{
                                                                volumeMode.label
                                                                }}</SelectItemText>
                                                            <span class="text-xs text-muted-foreground">
                                                                {{ volumeMode.desc }}
                                                            </span>
                                                        </div>
                                                    </div>
                                                </SelectItem>
                                            </SelectGroup>
                                        </SelectContent>
                                    </Select>
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
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
