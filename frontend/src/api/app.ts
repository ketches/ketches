import api from '@/api/axios'
import type { appInstanceModel, appModel, appRefModel, logsRequestModel } from '@/types/app'
import { toast } from 'vue-sonner'

export async function getApp(appID: string): Promise<appModel> {
    const response = await api.get(`/apps/${appID}`)
    return response.data as appModel
}

export async function getAppRef(appID: string): Promise<appRefModel> {
    const response = await api.get(`/apps/${appID}/ref`)
    return response.data as appRefModel
}

export async function appAction(appID: string, action: 'deploy' | 'start' | 'stop' | 'redeploy' | 'rollback' | 'rollingUpdate'): Promise<appModel> {
    const response = await api.post(`/apps/${appID}/action`, { action })
    return response.data as appModel
}

export async function deleteApp(appID: string): Promise<void> {
    await api.delete(`/apps/${appID}`)
    return
}

export async function listAppInstances(appID: string): Promise<{ deployVersion: string, instances: appInstanceModel[] }> {
    const response = await api.get(`/apps/${appID}/instances`)
    return response.data as { deployVersion: string, instances: appInstanceModel[] }
}

export async function terminateAppInstance(appID: string, instanceName: string): Promise<void> {
    await api.delete(`/apps/${appID}/instances`, { data: { instanceName } })
    return
}

export async function viewAppInstanceLogs(appID: string, instanceName: string, containerName: string, model?: logsRequestModel): Promise<string> {
    // SSE request to view app instance logs
    if (!model) {
        model = {
            follow: true,
            tailLines: 1000,
        }
    }
    const response = await api.get(`/apps/${appID}/instances/${instanceName}/containers/{${containerName}}/logs`, {
        params: {
            ...model
        }
    });
    return response.data as string;
}

export function getViewAppInstanceLogsUrl(appID: string, instanceName: string, containerName: string, params?: logsRequestModel) {
    if (!params) {
        params = {
            follow: true,
            tailLines: 1000,
        }
    }
    // Convert all values to strings for URLSearchParams compatibility
    const stringParams: Record<string, string> = {};
    for (const key in params) {
        if (Object.prototype.hasOwnProperty.call(params, key)) {
            stringParams[key] = String((params as any)[key]);
        }
    }
    const query = new URLSearchParams(stringParams).toString();
    const apibaseURL = import.meta.env.VITE_API_BASE_URL;
    return `${apibaseURL}/apps/${appID}/instances/${instanceName}/containers/${containerName}/logs?${query}`;
}

export function getExecAppInstanceTerminalUrl(appID: string, instanceName: string, containerName: string) {
    const apibaseURL = import.meta.env.VITE_API_BASE_URL;
    return `${apibaseURL}/apps/${appID}/instances/${instanceName}/containers/${containerName}/exec`;
}

export function appStatusToText(status: string): string {
    switch (status) {
        case 'undeployed':
            return '未部署'
        case 'starting':
            return '启动中'
        case 'running':
            return '运行中'
        case 'stopping':
            return '关闭中'
        case 'stopped':
            return '已关闭'
        case 'rollingUpdate':
            return '滚动更新中'
        case 'abnormal':
            return '异常'
        case 'completed':
            return '已完成'
        case 'unknown':
            return '未知'
        default:
            return status
    }
}

export async function exportApps(appIDs: string[]) {
    toast.info("Unimplemented!")
}
