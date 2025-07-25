export interface appModel {
    appID: string
    slug: string
    displayName: string
    description: string
    appType: string
    replicas: number
    containerImage: string
    containerCommand: string
    registryUsername: string
    registryPassword: string
    requestCPU: number
    requestMemory: number
    limitCPU: number
    limitMemory: number
    // Scheduling rules
    schedulingRule?: appSchedulingRuleModel
    edition: string
    envID: string
    projectID: string
    clusterID: string
    clusterNamespace: string
    actualReplicas: number
    actualEdition: string
    status: string
    createdAt: string
    updated: boolean
}

export interface appRefModel {
    appID: string
    slug: string
    displayName: string
    envID: string
    projectID: string
}

export interface createAppModel {
    slug: string
    displayName: string
    description?: string
    appType: string
    replicas: number
    containerImage: string
    registryUsername?: string
    registryPassword?: string
    requestCPU?: number
    requestMemory?: number
    limitCPU?: number
    limitMemory?: number
}

export interface updateAppInfoModel {
    displayName: string
    description?: string
}

export interface updateAppImageModel {
    containerImage: string
    registryUsername?: string
    registryPassword?: string
}

export interface setAppCommandModel {
    containerCommand: string
}

export interface setAppResourceModel {
    replicas: number
    requestCPU?: number
    requestMemory?: number
    limitCPU?: number
    limitMemory?: number
}

export interface appInstanceModel {
    instanceName: string
    status: string
    runningDuration: string
    instanceIP: string
    containerCount: number
    containers: appInstanceContainerModel[]
    initContainers: appInstanceContainerModel[]
    nodeName: string
    nodeIP: string
    edition: string
}

export interface appInstanceContainerModel {
    containerName: string
    status: string
}

export interface appRunningInfoModel {
    appID: string
    slug: string
    actualReplicas: number
    actualEdition: string
    status: string
    instances: appInstanceModel[]
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

export interface appVolumeModel {
    volumeID: string
    slug: string
    mountPath: string
    subPath?: string
    volumeMode?: string
    accessModes: string[]
    storageClass?: string
    capacity: number
    volumeType?: string
    appID: string
}

export interface appConfigFileModel {
    configFileID: string
    appID: string
    slug: string
    content: string
    mountPath: string
    fileMode: string
}

export interface createAppVolumeModel {
    slug: string
    mountPath: string
    subPath?: string
    volumeMode?: string
    accessModes: string[]
    storageClass?: string
    capacity: number
    volumeType?: string
}

export interface updateAppVolumeModel {
    mountPath: string
    subPath?: string
}

export interface appGatewayModel {
    gatewayID: string
    port: number
    protocol: string
    domain: string
    path: string
    certID: string
    gatewayPort: number
    exposed: boolean
    appID: string
}

export interface createAppGatewayModel {
    port: number
    protocol: string
    domain: string
    path: string
    certID: string
    gatewayPort: number
    exposed: boolean
}

export interface updateAppGatewayModel {
    port: number
    protocol: string
    domain: string
    path: string
    certID: string
    gatewayPort: number
    exposed: boolean
}

export interface appProbeModel {
    probeID: string
    type: 'liveness' | 'readiness' | 'startup'
    probeMode: 'httpGet' | 'tcpSocket' | 'exec'
    httpGetPort?: number
    httpGetPath?: string
    tcpSocketPort?: number
    execCommand?: string
    initialDelaySeconds: number
    periodSeconds: number
    timeoutSeconds: number
    successThreshold: number
    failureThreshold: number
    enabled: boolean
    appID: string
}

export interface createAppProbeModel {
    type: 'liveness' | 'readiness' | 'startup'
    probeMode: 'httpGet' | 'tcpSocket' | 'exec'
    httpGetPort?: number
    httpGetPath?: string
    tcpSocketPort?: number
    execCommand?: string
    initialDelaySeconds: number
    periodSeconds: number
    timeoutSeconds: number
    successThreshold: number
    failureThreshold: number
    enabled: boolean
}

export interface updateAppProbeModel {
    probeID: string
    type: 'liveness' | 'readiness' | 'startup'
    probeMode: 'httpGet' | 'tcpSocket' | 'exec'
    httpGetPort?: number
    httpGetPath?: string
    tcpSocketPort?: number
    execCommand?: string
    initialDelaySeconds: number
    periodSeconds: number
    timeoutSeconds: number
    successThreshold: number
    failureThreshold: number
    enabled: boolean
}

export interface appSchedulingRuleTolerationModel {
    key: string
    value: string
    operator: string // e.g., "Equal", "Exists"
    effect: string // e.g., "NoSchedule", "PreferNoSchedule", "NoExecute"
}

export interface appSchedulingRuleModel {
    ruleID: string
    appID: string
    ruleType: 'nodeName' | 'nodeSelector' | 'nodeAffinity' | ''
    nodeName?: string
    nodeSelector?: string[]
    nodeAffinity?: string[]
    tolerations?: appSchedulingRuleTolerationModel[]
}

export interface setAppSchedulingRuleModel {
    ruleType: 'nodeName' | 'nodeSelector' | 'nodeAffinity' | ''
    nodeName?: string
    nodeSelector?: string[]
    nodeAffinity?: string[]
    tolerations?: appSchedulingRuleTolerationModel[]
}