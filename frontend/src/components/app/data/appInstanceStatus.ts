import { Ban, Bug, CircleCheck, CircleQuestionMark, Clock, ClockAlert, ClockFading } from "lucide-vue-next";
import type { Component } from "vue";

export const appInstanceStatusEnum = {
    RUNNING: "running",
    PENDING: "pending",
    SUCCEEDED: "succeeded",
    TERMINATING: "terminating",
    ABNORMAL: "abnormal",
    DEBUGGING: "debugging",
    UNKNOWN: "unknown",
};

export interface appInstanceStatusDisplay {
    label: string;
    status: string;
    class: string;
    icon: Component;
}

export function appInstanceStatusDisplay(status: string): appInstanceStatusDisplay {
    switch (status) {
        case appInstanceStatusEnum.PENDING:
            return { label: "等待中", status: appInstanceStatusEnum.PENDING, class: "text-slate-600 dark:text-slate-400 bg-slate-100 dark:bg-slate-950 hover:bg-slate-100 dark:hover:bg-slate-950", icon: ClockFading };
        case appInstanceStatusEnum.RUNNING:
            return { label: "运行中", status: appInstanceStatusEnum.RUNNING, class: "text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-950 hover:bg-green-100 dark:hover:bg-green-950", icon: Clock };
        case appInstanceStatusEnum.TERMINATING:
            return { label: "终止中", status: appInstanceStatusEnum.TERMINATING, class: "text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-950 hover:bg-amber-100 dark:hover:bg-amber-950 ", icon: ClockAlert };
        case appInstanceStatusEnum.SUCCEEDED:
            return { label: "已完成", status: appInstanceStatusEnum.SUCCEEDED, class: "text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-950 hover:bg-green-100 dark:hover:bg-green-950", icon: CircleCheck };
        case appInstanceStatusEnum.ABNORMAL:
            return { label: "异常", status: appInstanceStatusEnum.ABNORMAL, class: "text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-950 hover:bg-red-100 dark:hover:bg-red-950", icon: ClockAlert };
        case appInstanceStatusEnum.DEBUGGING:
            return { label: "调试中", status: appInstanceStatusEnum.DEBUGGING, class: "text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-950 hover:bg-orange-100 dark:hover:bg-orange-950", icon: Bug };
        case appInstanceStatusEnum.UNKNOWN:
            return { label: "未知", status: appInstanceStatusEnum.UNKNOWN, class: "text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-950 hover:bg-gray-100 dark:hover:bg-gray-950", icon: Ban };
        default:
            return { label: status, status, class: "text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-950 hover:bg-gray-100 dark:hover:bg-gray-950", icon: CircleQuestionMark };
    }
}