import api from '@/api/axios';
import { useResourceRefStore } from '@/stores/resourceRefStore';
import { useUserStore } from '@/stores/userStore';
import type { QueryAndPagedRequest } from '@/types/common';
import type { userModel, userResourcesModel } from '@/types/user';

export async function signIn(username: string, password: string) {
    const response = await api.post('/users/sign-in', {
        username,
        password
    })
    const user = response.data as userModel
    useUserStore().setUser(user);
    return { success: true, data: user }
}

export async function signUp({
    username,
    fullname,
    email,
    password
}: {
    username: string
    fullname: string
    email: string
    password: string
}) {
    const response = await api.post('/users/sign-up', {
        username,
        fullname,
        email,
        password
    })
    return { success: true, data: response.data as userModel }
}


export async function signOut() {
    const userStore = useUserStore()
    const user = userStore.getUser()
    if (!user) {
        console.warn('No user found in localStorage, skipping sign-out')
        return { success: true }
    }
    await api.post('/users/sign-out', {
        "userID": user.userID
    })
    userStore.clearUser()
    useResourceRefStore().$reset() // Reset resource references
    return { success: true, data: user }
}

export async function listUsers(filter: QueryAndPagedRequest): Promise<{ total: number, records: userModel[] }> {
    const response = await api.get('/users', {
        params: filter
    })
    return response.data as { total: number, records: userModel[] }
}

export async function getUserInfo(userID: string) {
    const response = await api.get(`/users/${userID}`)
    const user = response.data as userModel
    return { success: true, data: user }
}

export async function fetchUserResourceRefs(): Promise<userResourcesModel> {
    const userStore = useUserStore()
    if (!userStore.getUser()) {
        await userStore.initUser()
    }

    const response = await api.get('/users/resources')
    return response.data as userResourcesModel
}
