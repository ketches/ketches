export interface projectModel {
    projectID: string
    slug: string
    displayName: string
    description: string
    createdAt: string
    updatedAt: string
}

export interface projectRefModel {
    projectID: string
    slug: string
    displayName: string
}

export interface createProjectModel {
    slug: string
    displayName: string
    description?: string
}


export interface projectMemberModel {
    projectID: string
    userID: string
    username: string
    email: string
    fullname: string
    phone: string
    projectRole: string
    createdAt: Date
}

