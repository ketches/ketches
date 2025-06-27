import { Braces, CircleCheckBig, Code, Cpu, Disc3, HardDrive, Library, Network, Radar, Rocket } from "lucide-vue-next"
import { defineAsyncComponent, type Component } from "vue"

interface Item {
    title: string
    tab: string
    icon?: Component
    comp?: Component
}

export const appSettingItems: Item[] = [
    {
        title: '基础信息',
        tab: 'appProfile',
        icon: Library,
        comp: defineAsyncComponent(() => import('../forms/AppProfile.vue')),
    },

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
