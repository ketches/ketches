import api from '@/api/axios'
import type { appEnvVarModel, appGatewayModel, appInstanceModel, appModel, appProbeModel, appRefModel, appSchedulingRuleModel, appVolumeModel, createAppEnvVarModel, createAppGatewayModel, createAppProbeModel, createAppVolumeModel, logsRequestModel, setAppCommandModel, setAppResourceModel, setAppSchedulingRuleModel, updateAppEnvVarModel, updateAppGatewayModel, updateAppImageModel, updateAppInfoModel, updateAppProbeModel, updateAppVolumeModel } from '@/types/app'
import { getApiBaseUrl } from '@/utils/env'
import { toast } from 'vue-sonner'

export async function getApp(appID: string): Promise<appModel> {
    const response = await api.get(`/apps/${appID}`)
    return response.data as appModel
}

export async function getAppRef(appID: string): Promise<appRefModel> {
    const response = await api.get(`/apps/${appID}/ref`)
    return response.data as appRefModel
}

export async function appAction(appID: string, action: string): Promise<appModel> {
    const response = await api.post(`/apps/${appID}/action`, { action })
    return response.data as appModel
}

export async function updateAppInfo(appID: string, model: updateAppInfoModel): Promise<appModel> {
    const response = await api.put(`/apps/${appID}`, model)
    return response.data as appModel
}

export async function deleteApp(appID: string): Promise<void> {
    await api.delete(`/apps/${appID}`)
    return
}

export async function updateAppImage(appID: string, model: updateAppImageModel): Promise<appModel> {
    const response = await api.put(`/apps/${appID}/image`, model)
    return response.data as appModel
}

export async function setAppCommand(appID: string, model: setAppCommandModel): Promise<appModel> {
    const response = await api.put(`/apps/${appID}/command`, model)
    return response.data as appModel
}

export async function setAppResource(appID: string, model: setAppResourceModel): Promise<appModel> {
    const response = await api.put(`/apps/${appID}/resource`, model)
    return response.data as appModel
}

export async function getAppSchedulingRule(appID: string): Promise<appSchedulingRuleModel> {
    const response = await api.get(`/apps/${appID}/scheduling-rule`)
    return response.data
}

export async function setAppSchedulingRule(appID: string, data: setAppSchedulingRuleModel): Promise<appSchedulingRuleModel> {
    const response = await api.put(`/apps/${appID}/scheduling-rule`, data)
    return response.data
}

export async function deleteAppSchedulingRule(appID: string): Promise<void> {
    await api.delete(`/apps/${appID}/scheduling-rule`)
    return
}

export async function listAppInstances(appID: string): Promise<{ edition: string, instances: appInstanceModel[] }> {
    const response = await api.get(`/apps/${appID}/instances`)
    return response.data as { edition: string, instances: appInstanceModel[] }
}

export function getAppRunningInfoUrl(appID: string) {
    const apibaseURL = getApiBaseUrl();
    return `${apibaseURL}/apps/${appID}/running/info`;
}

export async function terminateAppInstance(appID: string, instanceName: string): Promise<void> {
    await api.delete(`/apps/${appID}/instances`, { data: { instanceName } })
    return
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
    const apibaseURL = getApiBaseUrl();
    return `${apibaseURL}/apps/${appID}/instances/${instanceName}/containers/${containerName}/logs?${query}`;
}

export function getExecAppInstanceTerminalUrl(appID: string, instanceName: string, containerName: string) {
    const apibaseURL = getApiBaseUrl();
    return `${apibaseURL}/apps/${appID}/instances/${instanceName}/containers/${containerName}/exec`;
}

export async function listAppEnvVars(appID: string): Promise<appEnvVarModel[]> {
    const response = await api.get(`/apps/${appID}/env-vars`)
    return response.data as appEnvVarModel[]
}

export async function createAppEnvVar(appID: string, model: createAppEnvVarModel): Promise<appEnvVarModel> {
    const response = await api.post(`/apps/${appID}/env-vars`, model)
    return response.data as appEnvVarModel
}

export async function updateAppEnvVar(appID: string, envVarID: string, model: updateAppEnvVarModel): Promise<appEnvVarModel> {
    const response = await api.put(`/apps/${appID}/env-vars/${envVarID}`, model)
    return response.data as appEnvVarModel
}

export async function deleteAppEnvVar(appID: string, envVarID: string): Promise<void> {
    return deleteAppEnvVars(appID, [envVarID])
}

export async function deleteAppEnvVars(appID: string, envVarIDs: string[]): Promise<void> {
    await api.delete(`/apps/${appID}/env-vars`, {
        data: {
            envVarIDs: envVarIDs
        }
    })
    return
}

export async function listAppVolumes(appID: string): Promise<appVolumeModel[]> {
    const response = await api.get(`/apps/${appID}/volumes`)
    return response.data as appVolumeModel[] || []
}

export async function createAppVolume(appID: string, model: createAppVolumeModel): Promise<appVolumeModel> {
    const response = await api.post(`/apps/${appID}/volumes`, model)
    return response.data as appVolumeModel
}

export async function updateAppVolume(appID: string, volumeID: string, model: updateAppVolumeModel): Promise<appVolumeModel> {
    const response = await api.put(`/apps/${appID}/volumes/${volumeID}`, model)
    return response.data as appVolumeModel
}

export async function deleteAppVolume(appID: string, volumeID: string): Promise<void> {
    return deleteAppVolumes(appID, [volumeID])
}

export async function deleteAppVolumes(appID: string, volumeIDs: string[]): Promise<void> {
    await api.delete(`/apps/${appID}/volumes`, {
        data: {
            volumeIDs: volumeIDs
        }
    })
    return
}

export async function listAppGateways(appID: string): Promise<appGatewayModel[]> {
    const response = await api.get(`/apps/${appID}/gateways`);
    return response.data as appGatewayModel[];
}

export async function createAppGateway(appID: string, model: createAppGatewayModel): Promise<appGatewayModel> {
    const response = await api.post(`/apps/${appID}/gateways`, model);
    return response.data as appGatewayModel;
}

export async function updateAppGateway(appID: string, gatewayID: string, model: updateAppGatewayModel): Promise<appGatewayModel> {
    const response = await api.put(`/apps/${appID}/gateways/${gatewayID}`, model);
    return response.data as appGatewayModel;
}

export async function toggleAppGatewayExposed(appID: string, gatewayID: string, exposed: boolean): Promise<void> {
    await api.put(`/apps/${appID}/gateways/${gatewayID}/toggle`, { exposed });
    return
}

export async function deleteAppGateway(appID: string, gatewayIDs: string[]) {
    await api.delete(`/apps/${appID}/gateways`, {
        data: {
            gatewayIDs: gatewayIDs
        }
    });
}

export async function listAppProbes(appID: string): Promise<appProbeModel[]> {
    const response = await api.get(`/apps/${appID}/probes`);
    return response.data as appProbeModel[];
}

export async function createAppProbe(appID: string, model: createAppProbeModel): Promise<appProbeModel> {
    const response = await api.post(`/apps/${appID}/probes`, model);
    return response.data as appProbeModel;
}

export async function updateAppProbe(appID: string, probeID: string, model: updateAppProbeModel): Promise<appProbeModel> {
    const response = await api.put(`/apps/${appID}/probes/${probeID}`, model);
    return response.data as appProbeModel;
}

export async function toggleAppProbe(appID: string, probeID: string, enabled: boolean): Promise<appProbeModel> {
    const response = await api.put(`/apps/${appID}/probes/${probeID}/toggle`, { enabled });
    return response.data as appProbeModel;
}

export async function deleteAppProbe(appID: string, probeID: string): Promise<void> {
    await api.delete(`/apps/${appID}/probes/${probeID}`);
    return;
}

export async function exportApps(appIDs: string[]) {
    toast.info("Unimplemented!")
}
