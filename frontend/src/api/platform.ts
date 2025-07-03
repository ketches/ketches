import api from '@/api/axios';
import type { platformStatisticsModel } from '@/types/platform';

export async function fetchPlatformStatistics(): Promise<platformStatisticsModel> {
    const response = await api.get('/statistics')
    return response.data as platformStatisticsModel
}