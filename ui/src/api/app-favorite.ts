import client from './client';


export interface AppFavorite {
    id: string
    app_id: string
    env_id: string
}

export const appFavoritesApi = {
    listFavorites: (envId: string): Promise<AppFavorite[]> =>
        client.get(`/v1/envs/${envId}/apps/favorites`),
    getFavoriteStatus: (appId: string): Promise<{ is_favorite: boolean }> =>
        client.get(`/v1/apps/${appId}/favorite`),
    addFavorite: (appId: string): Promise<AppFavorite> =>
        client.post(`/v1/apps/${appId}/favorite`),
    removeFavorite: (appId: string): Promise<void> =>
        client.delete(`/v1/apps/${appId}/favorite`),
}
