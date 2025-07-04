import type { appRefModel } from "./app"
import type { clusterNodeRefModel, clusterRefModel } from "./cluster"
import type { envRefModel } from "./env"
import type { projectRefModel } from "./project"

export interface userModel {
    userID: string
    username: string
    email: string
    role: string
    fullname: string
    phone: string
    gender: number
    accessToken: string
    refreshToken: string
}

export interface userRefModel {
    userID: string
    username: string
    fullname: string
}

export interface userResourcesModel {
    projects: projectRefModel[]
    envs: envRefModel[]
    apps: appRefModel[]
}



export interface adminResourcesModel {
    clusters: clusterRefModel[]
    clusterNodes: clusterNodeRefModel[]
}