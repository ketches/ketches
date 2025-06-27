import { Ban, CircleCheck, CircleQuestionMark, Clock, ClockAlert, ClockFading } from "lucide-vue-next";
import type { Component } from "vue";

export const appInstanceStatusEnum = {
    RUNNING: "Running",
    PENDING: "Pending",
    SUCCEEDED: "Succeeded",
    FAILED: "Failed",
    UNKNOWN: "Unknown",
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
        case appInstanceStatusEnum.SUCCEEDED:
            return { label: "已完成", status: appInstanceStatusEnum.SUCCEEDED, fgColor: "text-green-600", icon: CircleCheck };
        case appInstanceStatusEnum.FAILED:
            return { label: "失败", status: appInstanceStatusEnum.FAILED, fgColor: "text-amber-600", icon: ClockAlert };
        case appInstanceStatusEnum.UNKNOWN:
            return { label: "未知", status: appInstanceStatusEnum.UNKNOWN, fgColor: "text-gray-600", icon: Ban };

        default:
            return { label: status, status, fgColor: "text-gray-600", icon: CircleQuestionMark };
    }
}