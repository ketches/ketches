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
    fgColor: string;
    icon: Component;
}

export function appInstanceStatusDisplay(status: string): appInstanceStatusDisplay {
    switch (status) {
        case appInstanceStatusEnum.PENDING:
            return { label: "等待中", status: appInstanceStatusEnum.PENDING, fgColor: "text-slate-600", icon: ClockFading };
        case appInstanceStatusEnum.RUNNING:
            return { label: "运行中", status: appInstanceStatusEnum.RUNNING, fgColor: "text-green-600", icon: Clock };
        case appInstanceStatusEnum.TERMINATING:
            return { label: "终止中", status: appInstanceStatusEnum.TERMINATING, fgColor: "text-amber-600", icon: ClockAlert };
        case appInstanceStatusEnum.SUCCEEDED:
            return { label: "已完成", status: appInstanceStatusEnum.SUCCEEDED, fgColor: "text-green-600", icon: CircleCheck };
        case appInstanceStatusEnum.ABNORMAL:
            return { label: "异常", status: appInstanceStatusEnum.ABNORMAL, fgColor: "text-amber-600", icon: ClockAlert };
        case appInstanceStatusEnum.DEBUGGING:
            return { label: "调试中", status: appInstanceStatusEnum.DEBUGGING, fgColor: "text-orange-600", icon: Bug };
        case appInstanceStatusEnum.UNKNOWN:
            return { label: "未知", status: appInstanceStatusEnum.UNKNOWN, fgColor: "text-gray-600", icon: Ban };

        default:
            return { label: status, status, fgColor: "text-gray-600", icon: CircleQuestionMark };
    }
}