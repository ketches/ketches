<script setup lang="ts">
import {
    createAppProbe,
    deleteAppProbe,
    listAppProbes,
    updateAppProbe,
} from "@/api/app";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import type { appModel, appProbeModel } from "@/types/app";
import { toTypedSchema } from "@vee-validate/zod";
import { CircleSlash, Edit, Save, Trash2 } from "lucide-vue-next";
import { useForm } from "vee-validate";
import { computed, onMounted, ref, toRef, watch } from "vue";
import { toast } from "vue-sonner";
import * as z from "zod";

const props = defineProps<{
    app: appModel;
}>();

const app = toRef(props, "app");

const probes = ref<appProbeModel[]>([]);

const isEditMode = ref({
    liveness: false,
    readiness: false,
    startup: false,
});

const probeTypes: Array<"liveness" | "readiness" | "startup"> = [
    "liveness",
    "readiness",
    "startup",
];

const probeDetails = {
    liveness: {
        title: "存活探针",
        description: "检测容器是否正常运行，如果检测失败，将自动重启容器。",
    },
    readiness: {
        title: "就绪探针",
        description: "检测容器是否正常运行，如果检测失败，将无法接收流量。",
    },
    startup: {
        title: "启动探针",
        description: "检测容器内应用是否已启动，在其成功之前，所有其他探针都将被禁用。",
    },
};

const getProbeByType = (type: "liveness" | "readiness" | "startup") => {
    return computed(() => probes.value.find((p) => p.type === type));
};

const livenessProbe = getProbeByType("liveness");
const readinessProbe = getProbeByType("readiness");
const startupProbe = getProbeByType("startup");

const probeRefs = {
    liveness: livenessProbe,
    readiness: readinessProbe,
    startup: startupProbe,
};

const formSchema = z.object({
    probeMode: z.enum(["httpGet", "tcpSocket", "exec"]),
    httpGetPort: z.coerce.number().min(1).max(65535).optional(),
    httpGetPath: z.string().startsWith("/").optional(),
    tcpSocketPort: z.coerce.number().min(1).max(65535).optional(),
    execCommand: z.string().optional(),
    initialDelaySeconds: z.coerce.number().min(0),
    periodSeconds: z.coerce.number().min(1),
    timeoutSeconds: z.coerce.number().min(1),
    successThreshold: z.coerce.number().min(1),
    failureThreshold: z.coerce.number().min(1),
});

const probeFormSchema = toTypedSchema(formSchema);

async function fetchProbes() {
    if (!app.value?.appID) return;
    try {
        probes.value = await listAppProbes(app.value.appID);
    } catch (error) {
        toast.error("获取探针信息失败");
    }
}

onMounted(fetchProbes);

watch(() => app.value.appID, fetchProbes);

type ProbeForm = z.infer<typeof formSchema>;

const { resetForm, values, setValues } = useForm<ProbeForm>({
    validationSchema: probeFormSchema,
});

function enterEditMode(type: "liveness" | "readiness" | "startup") {
    const probe = probeRefs[type].value;
    console.log("Entering edit mode for", type, "probe:", probe);

    let defaultInitialDelaySeconds = 5;
    if (type === "startup") {
        defaultInitialDelaySeconds = 30;
    }

    setValues(probe || {
        probeMode: "httpGet",
        httpGetPath: "/healthz",
        httpGetPort: 8080,
        tcpSocketPort: 8080,
        execCommand: "",
        initialDelaySeconds: defaultInitialDelaySeconds,
        periodSeconds: 10,
        timeoutSeconds: 5,
        successThreshold: 1,
        failureThreshold: 3,
    });
    isEditMode.value[type] = true;
}

function cancelEditMode(type: "liveness" | "readiness" | "startup") {
    resetForm();
    isEditMode.value[type] = false;
}

const onProbeSubmit = async (type: "liveness" | "readiness" | "startup") => {
    try {
        const existingProbe = probeRefs[type].value;
        if (existingProbe) {
            await updateAppProbe(app.value.appID, existingProbe.probeID, {
                probeID: existingProbe.probeID,
                type: type,
                probeMode: values.probeMode,
                httpGetPath: values.httpGetPath,
                httpGetPort: values.httpGetPort,
                tcpSocketPort: values.tcpSocketPort,
                execCommand: values.execCommand,
                initialDelaySeconds: values.initialDelaySeconds,
                periodSeconds: values.periodSeconds,
                timeoutSeconds: values.timeoutSeconds,
                successThreshold: values.successThreshold,
                failureThreshold: values.failureThreshold,
                enabled: existingProbe.enabled,
            });

            toast.success("探针更新成功");
        } else {
            console.log("Creating new probe with values:", values);
            await createAppProbe(app.value.appID, {
                type: type,
                probeMode: values.probeMode,
                httpGetPath: values.httpGetPath,
                httpGetPort: values.httpGetPort,
                tcpSocketPort: values.tcpSocketPort,
                execCommand: values.execCommand,
                initialDelaySeconds: values.initialDelaySeconds,
                periodSeconds: values.periodSeconds,
                timeoutSeconds: values.timeoutSeconds,
                successThreshold: values.successThreshold,
                failureThreshold: values.failureThreshold,
                enabled: true,
            });

            toast.success("探针创建成功");
        }
        await fetchProbes();
        isEditMode.value[type] = false;
    } catch (e) {
        console.error("Error in probe submission:", e);
        toast.error("提交探针信息时发生错误");
    }
}


async function toggleProbeEnabled(probe: appProbeModel) {
    await updateAppProbe(app.value.appID, probe.probeID, {
        ...probe,
        enabled: !probe.enabled,
    });
    toast.success(`探针已${!probe.enabled ? "启用" : "停用"}`);
    await fetchProbes();
}

async function handleDeleteProbe(probe: appProbeModel) {
    if (!confirm(`确定要删除 ${probeDetails[probe.type].title} 吗？`)) return;
    await deleteAppProbe(app.value.appID, probe.probeID);
    toast.success("探针删除成功");
    await fetchProbes();
    isEditMode.value[probe.type] = false;
}
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">容器探针</h3>
        <p class="text-sm text-muted-foreground">
            容器探针用于监控应用的运行状态。请确保您的应用能够响应探针请求。
        </p>
    </div>
    <Separator />
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <Card v-for="type in probeTypes" :key="type" :class="cn('w-full h-fit', $attrs.class ?? '')">
            <CardHeader>
                <CardTitle class="flex items-center">
                    <span>{{ probeDetails[type].title }}</span>
                    <div class="ml-auto flex items-center gap-2">
                        <Switch v-if="probeRefs[type].value" :checked="probeRefs[type].value?.enabled"
                            @update:checked="toggleProbeEnabled(probeRefs[type].value!)" />


                        <Button v-if="!isEditMode[type]" variant="ghost" size="icon" @click="enterEditMode(type)">
                            <Edit class="h-4 w-4" />
                        </Button>
                        <Button v-if="probeRefs[type].value" type="button" variant="ghost"
                            class="text-destructive hover:text-destructive ml-auto"
                            @click="handleDeleteProbe(probeRefs[type].value!)">
                            <Trash2 class="h-4 w-4" />
                        </Button>
                    </div>
                </CardTitle>
                <CardDescription>{{ probeDetails[type].description }}</CardDescription>
            </CardHeader>
            <CardContent class="grid gap-4">
                <!-- Edit Mode -->
                <form v-if="isEditMode[type]" class="space-y-4" @submit.prevent="onProbeSubmit(type)">
                    <FormField v-slot="{ value, handleChange }" name="probeMode">
                        <FormItem>
                            <FormControl>
                                <Select :model-value="value ?? 'httpGet'" @update:model-value="handleChange">
                                    <SelectTrigger class="w-full">
                                        <SelectValue placeholder="选择协议" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectGroup>
                                            <SelectItem value="httpGet">HTTP 探测</SelectItem>
                                            <SelectItem value="tcpSocket">TCP 探测</SelectItem>
                                            <SelectItem value="exec">执行命令探测</SelectItem>
                                        </SelectGroup>
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                    <div class="grid grid-cols-3 gap-4" v-if="values.probeMode === 'httpGet'">
                        <FormField v-slot="{ componentField }" name="httpGetPath">
                            <FormItem class="col-span-2">
                                <FormLabel>HTTP路径</FormLabel>
                                <FormControl>
                                    <Input type="text" placeholder="/healthz" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ componentField }" name="httpGetPort">
                            <FormItem class="col-span-1">
                                <FormLabel>端口</FormLabel>
                                <FormControl>
                                    <Input type="number" placeholder="8080" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div v-if="values.probeMode === 'tcpSocket'">
                        <FormField v-slot="{ componentField }" name="tcpSocketPort">
                            <FormItem>
                                <FormLabel>TCP 端口</FormLabel>
                                <FormControl>
                                    <Input type="number" placeholder="8080" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div v-if="values.probeMode === 'exec'">
                        <FormField v-slot="{ componentField }" name="execCommand">
                            <FormItem>
                                <FormLabel>执行命令</FormLabel>
                                <FormControl>
                                    <Input type="text" placeholder="cat /tmp/healthz.log" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <FormField v-slot="{ componentField }" name="initialDelaySeconds">
                        <FormItem class="col-span-1">
                            <FormLabel>初始延迟 (秒)</FormLabel>
                            <FormControl>
                                <Input type="number" v-bind="componentField" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    </FormField>
                    <div class="grid grid-cols-2 gap-4">
                        <FormField v-slot="{ componentField }" name="periodSeconds">
                            <FormItem class="col-span-1">
                                <FormLabel>检查周期 (秒)</FormLabel>
                                <FormControl>
                                    <Input type="number" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ componentField }" name="timeoutSeconds">
                            <FormItem class="col-span-1">
                                <FormLabel>超时时间 (秒)</FormLabel>
                                <FormControl>
                                    <Input type="number" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                        <FormField v-slot="{ componentField }" name="successThreshold">
                            <FormItem class="col-span-1">
                                <FormLabel>成功阈值</FormLabel>
                                <FormControl>
                                    <Input type="number" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                        <FormField v-slot="{ componentField }" name="failureThreshold">
                            <FormItem>
                                <FormLabel>失败阈值</FormLabel>
                                <FormControl>
                                    <Input type="number" v-bind="componentField" />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        </FormField>
                    </div>
                    <div class="flex gap-2 justify-start">
                        <Button type="button" variant="outline" @click="cancelEditMode(type)">
                            <CircleSlash />
                            取消
                        </Button>
                        <Button class="ml-auto" type="submit">
                            <Save />保存
                        </Button>
                    </div>
                </form>

                <!-- Read-only Mode -->
                <div v-else-if="probeRefs[type].value" class="space-y-2 text-sm text-muted-foreground">
                    <div v-if="probeRefs[type].value.probeMode === 'httpGet'">
                        <h4 class="text-big font-medium">
                            HTTP 探针
                        </h4>
                        <p class="text-sm text-muted-foreground">
                            HTTP 请求：<span class="font-mono">http://localhost:{{ probeRefs[type].value.httpGetPort
                            }}{{ probeRefs[type].value.httpGetPath }}</span>
                        </p>
                    </div>
                    <div v-else-if="probeRefs[type].value.probeMode === 'tcpSocket'">
                        <h4 class="text-big font-medium">TCP 探针</h4>
                        <p class="text-sm text-muted-foreground">
                            TCP 端口：<span class="font-mono">{{ probeRefs[type].value.tcpSocketPort }}</span>
                        </p>
                    </div>
                    <div v-else-if="probeRefs[type].value.probeMode === 'exec'">
                        <h4 class="text-big font-medium">执行命令探针</h4>
                        <p class="text-sm text-muted-foreground">
                            执行命令：<span class="font-mono">{{ probeRefs[type].value.execCommand }}</span>
                        </p>
                    </div>
                    <p v-if="type === 'liveness'">
                        在容器启动 {{ probeRefs[type].value?.initialDelaySeconds }} 秒后，每隔
                        {{ probeRefs[type].value?.periodSeconds }} 秒检测一次容器运行状态，
                        如果连续 {{ probeRefs[type].value?.failureThreshold }} 次检测失败，将重启容器。
                    </p>
                    <p v-if="type === 'readiness'">
                        在容器启动 {{ probeRefs[type].value?.initialDelaySeconds }} 秒后，每隔
                        {{ probeRefs[type].value?.periodSeconds }} 秒检测一次容器运行状态，
                        如果连续 {{ probeRefs[type].value?.failureThreshold }} 次检测失败，将无法接收流量。
                    </p>
                    <p v-if="type === 'startup'">
                        在容器启动 {{ probeRefs[type].value?.initialDelaySeconds }} 秒后，每隔
                        {{ probeRefs[type].value?.periodSeconds }} 秒检测一次容器运行状态，
                        如果连续 {{ probeRefs[type].value?.failureThreshold }} 次检测失败，将重启容器。
                        检测成功前，将不会启用其他探针。
                    </p>
                </div>
                <div v-else>
                    <p class="text-sm text-muted-foreground">未配置此探针。</p>
                </div>
            </CardContent>
        </Card>
    </div>
</template>
