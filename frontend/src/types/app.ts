export interface appModel {
    appID: string
    slug: string
    displayName: string
    description: string
    workloadType: string
    replicas: number
    containerImage: string
    registryUsername: string
    registryPassword: string
    requestCPU: number
    requestMemory: number
    limitCPU: number
    limitMemory: number
    deployed: boolean
    deployVersion: string
    envID: string
    projectID: string
    clusterID: string
    clusterNamespace: string
    actualReplicas: number
    status: string
    createdAt: string
}

export interface appRefModel {
    appID: string
    slug: string
    displayName: string
    envID: string
    projectID: string
}

export interface appCreateModel {
    slug: string
    displayName: string
    description?: string
    workloadType: string
    replicas: number
    containerImage: string
    registryUsername?: string
    registryPassword?: string
    requestCPU?: number
    requestMemory?: number
    limitCPU?: number
    limitMemory?: number
}

export interface appUpdateImageModel {
    containerImage: string
    registryUsername?: string
    registryPassword?: string
}

export interface appInstanceContainerModel {
    containerName: string
    status: string
}

export interface appInstanceModel {
    appID: string
    instanceName: string
    status: string
    runningDuration: string
    instanceIP: string
    containerCount: number
    containers: appInstanceContainerModel[]
    initContainers: appInstanceContainerModel[]
    nodeName: string
    nodeIP: string
    deployVersion: string
}

export interface logsRequestModel {
    follow?: boolean
    tailLines?: number
    showTimestamps?: boolean
    sinceSeconds?: number
    sinceTime?: string
    previous?: boolean
}

export interface appEnvVarModel {
    envVarID: string
    key: string
    value: string
    appID: string
}

export interface createAppEnvVarModel {
    key: string
    value: string
}

export interface updateAppEnvVarModel {
    value: string
}