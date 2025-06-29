import { Braces, CircleCheckBig, Code, Cpu, Disc3, HardDrive, Network, Radar, Rocket } from "lucide-vue-next"
import { defineAsyncComponent, type Component } from "vue"

interface Item {
    title: string
    tab: string
    icon?: Component
    comp?: Component
}

export const appSettingItems: Item[] = [
    {
        title: '容器镜像',
        tab: 'containerImage',
        icon: Disc3,
        comp: defineAsyncComponent(() => import('../forms/ContainerImage.vue')),
    },
    {
        title: '源码构建',
        tab: 'sourceCodeBuild',
        icon: Code,
        comp: defineAsyncComponent(() => import('../forms/SourceCodeBuild.vue')),
    },
    {
        title: '资源配置',
        tab: 'resourceConfig',
        icon: Cpu,
        comp: defineAsyncComponent(() => import('../forms/ResourceConfig.vue')),
    },
    {
        title: '环境变量',
        tab: 'envVar',
        icon: Braces,
        comp: defineAsyncComponent(() => import('../forms/EnvVar.vue')),
    },
    {
        title: '存储卷',
        tab: 'volume',
        icon: HardDrive,
        comp: defineAsyncComponent(() => import('../forms/Volume.vue')),
    },
    {
        title: '启动命令',
        tab: 'containerCommand',
        icon: Rocket,
        comp: defineAsyncComponent(() => import('../forms/ContainerCommand.vue')),
    },
    {
        title: '网关',
        tab: 'gateway',
        icon: Network,
        comp: defineAsyncComponent(() => import('../forms/Gateway.vue')),
    },
    {
        title: '健康检查',
        tab: 'healthCheck',
        icon: CircleCheckBig,
        comp: defineAsyncComponent(() => import('../forms/HealthCheck.vue')),
    },
    {
        title: '调度规则',
        tab: 'schedulingRule',
        icon: Radar,
        comp: defineAsyncComponent(() => import('../forms/SchedulingRule.vue')),
    },
]

export const appResourceSelectOptions = {
    cpu: [
        { label: '0.1 核', value: 100 },
        { label: '0.2 核', value: 200 },
        { label: '0.5 核', value: 500 },
        { label: '1 核', value: 1000 },
        { label: '2 核', value: 2000 },
        { label: '4 核', value: 4000 },
        { label: '8 核', value: 8000 },
        { label: '16 核', value: 16000 },
        { label: '32 核', value: 32000 },
    ],
    memory: [
        { label: '128 MiB', value: 128 },
        { label: '256 MiB', value: 256 },
        { label: '512 MiB', value: 512 },
        { label: '1 GiB', value: 1024 },
        { label: '2 GiB', value: 2048 },
        { label: '4 GiB', value: 4096 },
        { label: '8 GiB', value: 8192 },
        { label: '16 GiB', value: 16384 },
        { label: '32 GiB', value: 32768 },
        { label: '64 GiB', value: 65536 },
    ],
}