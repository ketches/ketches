export interface envModel {
    envID: string
    slug: string
    displayName: string
    description: string
    projectID: string
    clusterID: string
    createdAt: string
}

export interface envRefModel {
    envID: string
    slug: string
    displayName: string
    projectID: string
}

export interface envCreateModel {
    projectID: string
    slug: string
    displayName: string
    description?: string | ""
    clusterID: string | ""
}

export interface updateEnvModel {
    displayName: string,
    description?: string,
}

