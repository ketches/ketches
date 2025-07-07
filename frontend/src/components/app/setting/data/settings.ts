import {
  Braces,
  Code,
  Cpu,
  Disc3,
  FolderTree,
  HardDrive,
  HeartPulse,
  Network,
  Radar,
  Rocket,
  Server,
  SquareDashed,
  SquaresExclude,
  SquaresIntersect,
  SquaresSubtract,
  Tag,
  Vault
} from "lucide-vue-next";
import { defineAsyncComponent, type Component } from "vue";

interface Item {
  title: string;
  tab: string;
  icon?: Component;
  comp?: Component;
}

export const appSettingItems: Item[] = [
  {
    title: "容器镜像",
    tab: "containerImage",
    icon: Disc3,
    comp: defineAsyncComponent(() => import("../forms/ContainerImage.vue")),
  },
  {
    title: "源码构建",
    tab: "sourceCodeBuild",
    icon: Code,
    comp: defineAsyncComponent(() => import("../forms/SourceCodeBuild.vue")),
  },
  {
    title: "启动命令",
    tab: "containerCommand",
    icon: Rocket,
    comp: defineAsyncComponent(() => import("../forms/ContainerCommand.vue")),
  },
  {
    title: "资源配置",
    tab: "resourceConfig",
    icon: Cpu,
    comp: defineAsyncComponent(() => import("../forms/ResourceConfig.vue")),
  },
  {
    title: "环境变量",
    tab: "envVar",
    icon: Braces,
    comp: defineAsyncComponent(() => import("../forms/EnvVar.vue")),
  },
  {
    title: "存储卷",
    tab: "volume",
    icon: HardDrive,
    comp: defineAsyncComponent(() => import("../forms/Volume.vue")),
  },
  {
    title: "端口网关",
    tab: "gateway",
    icon: Network,
    comp: defineAsyncComponent(() => import("../forms/Gateway.vue")),
  },
  {
    title: "容器探针",
    tab: "probe",
    icon: HeartPulse,
    comp: defineAsyncComponent(() => import("../forms/Probe.vue")),
  },
  {
    title: "调度规则",
    tab: "schedulingRule",
    icon: Radar,
    comp: defineAsyncComponent(() => import("../forms/SchedulingRule.vue")),
  },
];

export const appResourceSelectOptions = {
  cpu: [
    { label: "0.1 核", value: 100 },
    { label: "0.2 核", value: 200 },
    { label: "0.5 核", value: 500 },
    { label: "1 核", value: 1000 },
    { label: "2 核", value: 2000 },
    { label: "4 核", value: 4000 },
    { label: "8 核", value: 8000 },
    { label: "16 核", value: 16000 },
    { label: "32 核", value: 32000 },
  ],
  memory: [
    { label: "128 MiB", value: 128 },
    { label: "256 MiB", value: 256 },
    { label: "512 MiB", value: 512 },
    { label: "1 GiB", value: 1024 },
    { label: "2 GiB", value: 2048 },
    { label: "4 GiB", value: 4096 },
    { label: "8 GiB", value: 8192 },
    { label: "16 GiB", value: 16384 },
    { label: "32 GiB", value: 32768 },
    { label: "64 GiB", value: 65536 },
  ],
};

export const volumeTypeRefs = {
  pvc: {
    label: "持久化存储",
    value: "pvc",
    icon: HardDrive,
    desc: "数据将被持久化，实例重启或漂移后数据不丢失。",
  },
  emptyDir: {
    label: "临时存储",
    value: "emptyDir",
    icon: SquaresIntersect,
    desc: "多容器共享的临时缓存，实例重启后数据丢失。",
  },
  hostPath: {
    label: "本地存储",
    value: "hostPath",
    icon: SquaresSubtract,
    desc: "节点本地存储，实例漂移后数据可能丢失。",
  },
};

export const accessModeRefs = {
  ReadWriteOnce: {
    label: "单节点读写",
    value: "ReadWriteOnce",
    icon: SquareDashed,
    desc: "存储卷只能被单个节点挂载为读写模式。",
  },
  ReadWriteMany: {
    label: "多节点读写",
    value: "ReadWriteMany",
    icon: SquaresExclude,
    desc: "存储卷可以被多个节点同时挂载为读写模式。",
  },
  ReadOnlyMany: {
    label: "多节点只读",
    value: "ReadOnlyMany",
    icon: SquaresIntersect,
    desc: "存储卷可以被多个节点挂载为只读模式。",
  },
};

export const volumeModeRefs = {
  Filesystem: {
    label: "文件系统",
    value: "Filesystem",
    icon: FolderTree,
    desc: "存储卷以文件系统方式挂载。",
  },
  Block: {
    label: "块存储",
    value: "Block",
    icon: Vault,
    desc: "存储卷以块设备方式挂载。",
  },
};

export const schedulingRuleTypeRefs = {
  nodeName: {
    label: "指定节点名称",
    value: "nodeName",
    icon: Server,
    desc: "将应用实例调度到指定节点上运行。",
  },
  nodeSelector: {
    label: "节点标签选择器",
    value: "nodeSelector",
    icon: Tag,
    desc: "将应用实例调度到符合标签选择器的节点上运行。",
  },
  nodeAffinity: {
    label: "节点亲和性",
    value: "nodeAffinity",
    icon: Server,
    desc: "将应用实例调度到亲和节点上运行。",
  },
}

export const schedulingRuleTolerationOperatorRefs = {
  Equal: {
    label: "等于",
    value: "Equal",
  },
  Exists: {
    label: "存在",
    value: "Exists",
  },
};

export const schedulingRuleTolerationEffectRefs = {
  NoSchedule: {
    label: "不允许调度",
    value: "NoSchedule",
  },
  PreferNoSchedule: {
    label: "尽量阻止调度",
    value: "PreferNoSchedule",
  },
  NoExecute: {
    label: "阻止调度并驱逐已调度",
    value: "NoExecute",
  },
};
