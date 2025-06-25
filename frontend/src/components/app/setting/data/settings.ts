import { Box, Braces, CircleCheckBig, Code, HardDrive, Network, Radar } from "lucide-vue-next"
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
        icon: Box,
        comp: defineAsyncComponent(() => import('../forms/ContainerImage.vue')),
    },
    {
        title: '源码构建',
        tab: 'sourceCodeBuild',
        icon: Code,
        comp: defineAsyncComponent(() => import('../forms/SourceCodeBuild.vue')),
    },
    {
        title: '环境变量',
        tab: 'envVar',
        icon: Braces,
        comp: defineAsyncComponent(() => import('../forms/EnvVar.vue')),
    },
    {
        title: '持久化存储',
        tab: 'volume',
        icon: HardDrive,
        comp: defineAsyncComponent(() => import('../forms/Volume.vue')),
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
