import { appAction } from "@/api/app";
import type { appModel } from "@/types/app";
import { ArrowBigUpDash, Bug, BugOff, CircleAlert, CircleQuestionMark, CloudCog, Package, PackageCheck, PackageMinus, Play, Power, RefreshCcwDot, Replace, Rocket } from "lucide-vue-next";
import type { Component } from "vue";
import { toast } from "vue-sonner";

export const appStatusEnum = {
    UNDEPLOYED: "undeployed",
    STARTING: "starting",
    RUNNING: "running",
    STOPPING: "stopping",
    STOPPED: "stopped",
    UPDATING: "updating",
    ABNORMAL: "abnormal",
    COMPLETED: "completed",
    DEBUGGING: "debugging",
    UNKNOWN: "unknown",
};

export interface appStatusDisplay {
    label: string;
    status: string;
    fgColor: string;
    icon: Component;
}

export function appStatusDisplay(status: string): appStatusDisplay {
    switch (status) {
        case appStatusEnum.UNDEPLOYED:
            return { label: "未部署", status: appStatusEnum.UNDEPLOYED, fgColor: "text-gray-600", icon: PackageMinus };
        case appStatusEnum.STARTING:
            return { label: "启动中", status: appStatusEnum.STARTING, fgColor: "text-amber-600", icon: Rocket };
        case appStatusEnum.RUNNING:
            return { label: "运行中", status: appStatusEnum.RUNNING, fgColor: "text-green-600", icon: PackageCheck };
        case appStatusEnum.STOPPING:
            return { label: "关闭中", status: appStatusEnum.STOPPING, fgColor: "text-amber-600", icon: PackageMinus };
        case appStatusEnum.STOPPED:
            return { label: "已关闭", status: appStatusEnum.STOPPED, fgColor: "text-gray-600", icon: PackageMinus };
        case appStatusEnum.UPDATING:
            return { label: "更新中", status: appStatusEnum.UPDATING, fgColor: "text-amber-600", icon: Package };
        case appStatusEnum.ABNORMAL:
            return { label: "异常", status: appStatusEnum.ABNORMAL, fgColor: "text-red-600", icon: CircleAlert };
        case appStatusEnum.COMPLETED:
            return { label: "已完成", status: appStatusEnum.COMPLETED, fgColor: "text-green-600", icon: PackageCheck };
        case appStatusEnum.DEBUGGING:
            return { label: "调试中", status: appStatusEnum.DEBUGGING, fgColor: "text-orange-600", icon: Bug };
        case appStatusEnum.UNKNOWN:
            return { label: "未知", status: appStatusEnum.UNKNOWN, fgColor: "text-gray-600", icon: CircleQuestionMark };
        default:
            return { label: status, status, fgColor: "text-gray-600", icon: CircleQuestionMark };
    }
}

export interface appStatusAction {
    label: string;
    action: (appID: string) => Promise<appModel> | Promise<void>;
    icon: Component;
    tip?: boolean;
}

export function appStatusActions(appStatus: string, desiredEdition?: string, actualEdition?: string): appStatusAction[] {
    const actions: appStatusAction[] = [];
    switch (appStatus) {
        case appStatusEnum.UNDEPLOYED:
            actions.push({ label: "部署", icon: CloudCog, action: onDeploy });
            break;
        case appStatusEnum.STOPPED:
            actions.push({ label: "启动", icon: Play, action: onStart });
            break;
        case appStatusEnum.STARTING:
        case appStatusEnum.UPDATING:
        case appStatusEnum.ABNORMAL: {
            let updateAction: appStatusAction = { label: "更新", icon: ArrowBigUpDash, action: onUpdate };
            if (desiredEdition && actualEdition && desiredEdition !== actualEdition) {
                updateAction.tip = true
            }
            actions.push(updateAction);
            actions.push(
                { label: "关闭", icon: Power, action: onStop },
                { label: "重新部署", icon: Replace, action: onRedeploy },
            );
            break;
        }
        case appStatusEnum.RUNNING: {
            let updateAction: appStatusAction = { label: "更新", icon: ArrowBigUpDash, action: onUpdate };
            if (desiredEdition && actualEdition && desiredEdition !== actualEdition) {
                updateAction.tip = true
            }
            actions.push(updateAction);
            actions.push(
                { label: "关闭", icon: Power, action: onStop },
                { label: "重新部署", icon: RefreshCcwDot, action: onRedeploy },
            );
            break;
        }
        case appStatusEnum.COMPLETED:
            actions.push(
                { label: "关闭", icon: Power, action: onStop },
                { label: "重新部署", icon: Replace, action: onRedeploy }
            );
            break;
        case appStatusEnum.DEBUGGING:
            actions.push(
                { label: "退出调试", icon: BugOff, action: onDebugOff },
            );
            break;
        case "error":
        default:
            break;
    }
    return actions;
}

async function onDeploy(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "deploy")
    toast.success("部署成功！", {
        description: `应用 ${resp.slug} 已成功部署。`,
    });
    return resp
}

async function onRedeploy(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "redeploy")
    toast.success("应用正在重新部署", {
        description: `应用 ${resp.slug} 正在重新部署。`,
    });
    return resp
}

async function onStart(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "start")
    toast.success("应用正在启动", {
        description: `应用 ${resp.slug} 正在启动。`,
    });
    return resp
}

async function onStop(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "stop")
    toast.success("应用正在关闭", {
        description: `应用 ${resp.slug} 正在关闭。`,
    });
    return resp
}

async function onUpdate(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "update")
    toast.success("应用正在更新", {
        description: `应用 ${resp.slug} 正在进行滚动更新。`,
    });
    return resp
}

async function onDebug(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "debug")
    toast.success("应用开始进入调试", {
        description: `等待应用 ${resp.slug} 实例重启完成后，您可以进入实例终端进行调试操作。`,
    });
    return resp
}

async function onDebugOff(appID: string): Promise<appModel> {
    const resp = await appAction(appID, "debugOff")
    toast.success("应用正在退出调试", {
        description: `应用 ${resp.slug} 正在退出调试模式。`,
    });
    return resp
}

async function onDelete(appID: string): Promise<void> {
    const resp = await appAction(appID, "delete")
    toast.success("应用已删除", {
        description: `应用 ${resp.slug} 已成功删除。`,
    });
    return
}
