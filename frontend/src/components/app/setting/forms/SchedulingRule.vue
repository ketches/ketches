<script setup lang="ts">
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { onMounted, ref, toRef, watch } from 'vue';
import * as z from 'zod';

import { deleteAppSchedulingRule, getAppSchedulingRule, setAppSchedulingRule } from '@/api/app';
import { listClusterNodeLabels, listClusterNodeRefs, listClusterNodeTaints } from '@/api/cluster';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectItemText,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { appModel, appSchedulingRuleModel, setAppSchedulingRuleModel } from '@/types/app';
import type { clusterNodeRefModel, clusterNodeTaintsModel } from '@/types/cluster';
import { Delete, Loader, Plus, Save, Server, Tag, X } from 'lucide-vue-next';
import { toast } from 'vue-sonner';
import { schedulingRuleTolerationEffectRefs, schedulingRuleTolerationOperatorRefs, schedulingRuleTypeRefs } from '../data/settings';


const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = toRef(props, "app");

const schedulingRuleFormSchema = toTypedSchema(z.object({
    ruleType: z.enum(['nodeName', 'nodeSelector', 'nodeAffinity', '']).optional(),
    targetNodeName: z.string().optional(),
    nodeSelector: z.array(z.string()).optional(),
    nodeAffinity: z.array(z.string()).optional(),
}))

const { handleSubmit, resetForm, setValues, values } = useForm({
    validationSchema: schedulingRuleFormSchema,
    initialValues: {
        ruleType: undefined,
        targetNodeName: '',
        nodeSelector: [],
        nodeAffinity: [],
    },
})

const loading = ref(false)
const nodeOptions = ref<clusterNodeRefModel[]>([])
const nodeLabelOptions = ref<string[]>([])
const nodeTaintsOptions = ref<clusterNodeTaintsModel[]>([])

// 加载节点和标签选项
const loadOptions = async () => {
    if (!app.value) return

    try {
        // 获取当前应用所在集群的节点列表
        const clusterID = app.value.clusterID
        if (clusterID) {
            const [nodes, labels, taints] = await Promise.all([
                listClusterNodeRefs(clusterID),
                listClusterNodeLabels(clusterID),
                listClusterNodeTaints(clusterID)
            ])
            nodeOptions.value = nodes
            nodeLabelOptions.value = labels
            nodeTaintsOptions.value = taints
        }
    } catch (error) {
        console.error('加载选项失败:', error)
    }
}

const rule = ref<appSchedulingRuleModel>({
    ruleID: '',
    appID: '',
    ruleType: 'nodeName',
    tolerations: [],
})

// 加载调度规则数据
const loadSchedulingRule = async () => {
    if (!app.value) return

    try {
        loading.value = true
        rule.value = await getAppSchedulingRule(app.value.appID)

        if (rule.value) {
            setValues({
                ruleType: rule.value.ruleType,
                targetNodeName: rule.value.nodeName || '',
                nodeSelector: rule.value.nodeSelector,
                nodeAffinity: rule.value.nodeAffinity || [],
            })
        }
    } catch (error: any) {
        console.error('加载调度规则失败:', error)
    } finally {
        loading.value = false
    }
}

function addToleration() {
    if (!rule.value) {
        // 新建 rule 并添加 tolerations
        rule.value = {
            ruleID: '',
            appID: app.value?.appID || '',
            ruleType: 'nodeName',
            tolerations: [{ key: '', value: '', operator: 'Equal', effect: 'NoSchedule' }]
        };
    } else {
        if (!Array.isArray(rule.value.tolerations)) rule.value.tolerations = [];
        rule.value.tolerations.push({ key: '', value: '', operator: 'Equal', effect: 'NoSchedule' });
    }
}

// 监听当前应用变化
watch(app, (newApp) => {
    if (newApp) {
        loadOptions()
        loadSchedulingRule()
    }
})

// 组件挂载时加载数据
onMounted(() => {
    if (app.value) {
        loadOptions()
        loadSchedulingRule()
    }
})

const onSubmit = handleSubmit(async (formValues) => {
    if (!app.value) {
        toast.error('未找到当前应用')
        return
    }

    try {
        loading.value = true

        const requestData: setAppSchedulingRuleModel = {
            ruleType: formValues.ruleType,
        }

        if (formValues.ruleType === 'nodeName') {
            if (!formValues.targetNodeName) {
                toast.error('请输入节点名称')
                return
            }
            requestData.nodeName = formValues.targetNodeName
        } else if (formValues.ruleType === 'nodeSelector') {
            if (formValues.nodeSelector.length === 0) {
                toast.error('请添加至少一个节点标签')
                return
            }
            requestData.nodeSelector = formValues.nodeSelector
        } else if (formValues.ruleType === 'nodeAffinity') {
            if (!formValues.nodeAffinity || formValues.nodeAffinity.length === 0) {
                toast.error('请选择至少一个节点')
                return
            }
            requestData.nodeAffinity = formValues.nodeAffinity
        }

        // 新增：提交tolerations
        if (Array.isArray(rule.value.tolerations)) {
            for (const tol of rule.value.tolerations) {
                if (!tol.key && tol.operator === 'Equal') {
                    toast.error('当前容忍设置的操作符为 Equal 时，容忍键不能为空')
                    return
                }
                if (!tol.key && tol.value) {
                    toast.error('当容忍键为空时，不能设置容忍值')
                    return
                }
            }
            requestData.tolerations = rule.value.tolerations;
        }

        await setAppSchedulingRule(app.value.appID, requestData)
        toast.success('调度规则设置成功')
        app.value.updated = true

        await loadSchedulingRule()
    } catch (error: any) {
        toast.error(error.message || '设置调度规则失败')
    } finally {
        loading.value = false
    }
})

const onDelete = async () => {
    if (!app.value) {
        toast.error('未找到当前应用')
        return
    }

    try {
        loading.value = true
        await deleteAppSchedulingRule(app.value.appID)
        toast.success('调度规则删除成功')
        rule.value = null
        app.value.updated = true

        // 重置表单
        resetForm()
    } catch (error: any) {
        toast.error(error.message || '删除调度规则失败')
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            调度规则
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用应用实例在集群中的调度位置。
        </p>
    </div>
    <Separator />

    <form class="space-y-8" @submit="onSubmit">
        <div class="grid grid-cols-3 gap-4">
            <FormField v-slot="{ value, handleChange }" name="ruleType">
                <FormItem class="col-span-1">
                    <FormLabel>调度规则类型</FormLabel>
                    <Select :model-value="value" @update:model-value="handleChange">
                        <FormControl>
                            <SelectTrigger class="w-full">
                                <SelectValue>
                                    <div v-if="value" class="flex items-center">
                                        <component :is="schedulingRuleTypeRefs[
                                            value as keyof typeof schedulingRuleTypeRefs
                                        ]?.icon" class="h-4 w-4 mr-2" />
                                        <span>{{ schedulingRuleTypeRefs[value as keyof typeof
                                            schedulingRuleTypeRefs]?.label
                                        }}</span>
                                    </div>
                                    <span v-else>选择调度规则类型</span>
                                </SelectValue>
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="(schedulingRuleType, key) in schedulingRuleTypeRefs" :key="key"
                                    :value="key">
                                    <div class="flex items-center gap-3">
                                        <component :is="schedulingRuleType.icon" class="h-4 w-4" />
                                        <div class="flex flex-col">
                                            <SelectItemText>{{
                                                schedulingRuleType.label
                                                }}</SelectItemText>
                                            <span class="text-xs text-muted-foreground">{{
                                                schedulingRuleType.desc
                                                }}</span>
                                        </div>
                                    </div>
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                    <FormDescription>
                        选择调度规则类型，决定应用实例在集群中的调度方式。
                    </FormDescription>
                    <FormMessage />
                </FormItem>
            </FormField>

            <FormField v-if="values.ruleType === 'nodeName'" v-slot="{ value, handleChange }" name="targetNodeName">
                <FormItem class="col-span-2">
                    <FormLabel>节点名称</FormLabel>
                    <Select :model-value="value" @update:model-value="handleChange">
                        <FormControl>
                            <SelectTrigger class="w-full">
                                <SelectValue>
                                    <div v-if="value" class="flex items-center">
                                        <Server class="h-4 w-4 mr-3" />
                                        {{ value }}
                                    </div>
                                    <span v-else>选择节点</span>
                                </SelectValue>
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="node in nodeOptions" :key="node.nodeName" :value="node.nodeName">
                                    <div class="flex items-center gap-3">
                                        <Server class="h-4 w-4" />
                                        <div class="flex flex-col">
                                            <SelectItemText>{{
                                                node.nodeName
                                            }}</SelectItemText>
                                            <span class="text-xs text-muted-foreground">{{
                                                node.nodeIP
                                            }}</span>
                                        </div>
                                    </div>
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                    <FormDescription>
                        将应用实例调度到指定节点上运行。
                    </FormDescription>
                    <FormMessage />
                </FormItem>
            </FormField>

            <FormField v-if="values.ruleType === 'nodeSelector'" v-slot="{ componentField }" name="nodeSelector">
                <FormItem class="col-span-2">
                    <FormLabel>节点选择器</FormLabel>
                    <Select v-bind="componentField" multiple>
                        <FormControl>
                            <SelectTrigger class="w-full">
                                <SelectValue placeholder="选择节点标签，允许多个" />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="label in nodeLabelOptions" :key="label" :value="label">
                                    <Tag class="h-4 w-4" />
                                    <div class="flex flex-col">
                                        <SelectItemText>{{
                                            label
                                            }}</SelectItemText>
                                    </div>
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                    <FormDescription>
                        将应用实例调度到符合标签选择器的节点上运行。
                    </FormDescription>
                    <FormMessage />
                </FormItem>
            </FormField>
            <FormField v-if="values.ruleType === 'nodeAffinity'" v-slot="{ componentField }" name="nodeAffinity">
                <FormItem class="col-span-2">
                    <FormLabel>节点亲和性</FormLabel>
                    <Select v-bind="componentField" multiple>
                        <FormControl>
                            <SelectTrigger class="w-full">
                                <SelectValue placeholder="选择节点，允许多个" />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem v-for="node in nodeOptions" :key="node.nodeName" :value="node.nodeName">
                                    <Server class="h-4 w-4" />
                                    <div class="flex flex-col">
                                        <SelectItemText>{{
                                            node.nodeName
                                            }}</SelectItemText>
                                        <span class="text-xs text-muted-foreground">{{
                                            node.nodeIP
                                            }}</span>
                                    </div>
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                    <FormDescription>
                        将应用实例调度到亲和节点上运行。
                    </FormDescription>
                    <FormMessage />
                </FormItem>
            </FormField>
        </div>
        <Separator />
        <div class="flex flex-col flex-grow">
            <div class="flex gap-2 items-center pb-4 w-full">
                <Label class="text-sm font-medium">污点容忍设置</Label>
                <Button variant="outline" size="sm" class="ml-auto" @click.prevent="addToleration">
                    <Plus />
                    新增
                </Button>
            </div>
            <div class="space-y-2">
                <div v-for="(tol, idx) in (rule && rule.tolerations ? rule.tolerations : [])" :key="idx"
                    class="flex gap-2 items-center">
                    <Select v-model="tol.key" class="flex-grow">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="容忍污点键" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem v-for="taints in nodeTaintsOptions" :key="taints.key" :value="taints.key">
                                <div class="flex flex-col">
                                    <SelectItemText>{{
                                        taints.key
                                    }}</SelectItemText>
                                </div>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                    <Select v-model="tol.operator" class="flex-grow">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="操作符" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem v-for="operator in schedulingRuleTolerationOperatorRefs" :key="operator.value"
                                :value="operator.value">
                                <div class="flex flex-col">
                                    <SelectItemText>{{
                                        operator.label
                                    }}</SelectItemText>
                                    <span class="text-xs text-muted-foreground">{{
                                        operator.value
                                    }}</span>
                                </div>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                    <Select v-model="tol.value" class="flex-grow">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="容忍污点值" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem
                                v-for="taintValue in (nodeTaintsOptions.find(taints => taints.key === tol.key)?.values || [])"
                                :key="taintValue" :value="taintValue">
                                <div class="flex flex-col">
                                    <SelectItemText>{{
                                        taintValue
                                    }}</SelectItemText>
                                </div>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                    <Select v-model="tol.effect" class="flex-grow">
                        <SelectTrigger class="w-full">
                            <SelectValue placeholder="效果" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem v-for="effect in schedulingRuleTolerationEffectRefs" :key="effect.value"
                                :value="effect.value">
                                <div class="flex flex-col">
                                    <SelectItemText>{{
                                        effect.label
                                    }}</SelectItemText>
                                    <span class="text-xs text-muted-foreground">{{
                                        effect.value
                                    }}</span>
                                </div>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                    <Button variant="outline" size="icon"
                        @click.prevent="() => rule && rule && rule.tolerations && rule.tolerations.splice(idx, 1)"
                        class="flex-shrink-0 rounded-full text-destructive hover:text-destructive hover:bg-destructive/10">
                        <X />
                    </Button>
                </div>
            </div>
        </div>
        <div class="flex gap-2 justify-start">
            <Button type="submit">
                <Loader v-if="loading" />
                <Save v-else />
                保存
            </Button>
            <Button v-if="rule" type="button" variant="destructive" @click="onDelete" :disabled="loading">
                <Delete />
                清除
            </Button>
        </div>
    </form>
</template>
