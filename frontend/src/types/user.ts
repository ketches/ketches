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