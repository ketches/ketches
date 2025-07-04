import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
// If useRouter() causes issues at the top level, import the router instance directly:
// import router from '@/router'; // Assuming your router is configured and exported
import { useUserStore } from '@/stores/userStore'; // Adjust the import path as necessary
import { getApiBaseUrl } from '@/utils/env';


const apibaseURL = getApiBaseUrl();

const instance = axios.create({
    baseURL: apibaseURL,
    timeout: 5000,
});

// Request Interceptor
instance.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
        config.withCredentials = true; // Send cookies with requests
        return config;
    },
    (error: AxiosError) => {
        return Promise.reject<AxiosError>(error);
    }
);

// Response Interceptor
instance.interceptors.response.use(
    (response) => {
        // If status is 200-299, return response.data directly
        if (response.status >= 200 && response.status < 300) {
            return response.data
        }
        // For other non-error statuses (e.g., 3xx), axios might handle them or pass through.
        // This ensures that if axios doesn't throw an error for a non-2xx status,
        // the full response is passed along unless explicitly handled.
        return response;
    },
    async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

        if (error.response) {
            const status = error.response.status;
            const responseData = error.response.data as any; // Type assertion for responseData

            if (status === 401) {
                const userStore = useUserStore();
                if (originalRequest.url === '/users/refresh-token') {
                    // 401 but already on /users/refresh-token
                    userStore.clearUser();

                    let signInUrl = '/sign-in';
                    const redirectUrl = window.location.search + window.location.hash
                    if (redirectUrl !== "" && redirectUrl !== "/") {
                        signInUrl += '?redirectUrl=' + encodeURIComponent(redirectUrl);
                    }
                    window.location.href = signInUrl
                    return Promise.reject(error);
                }

                // Retry to refresh token
                const refreshTokenResponse = await instance.post('/users/refresh-token');
                if (refreshTokenResponse.data.userID) { // Check custom success code
                    userStore.setUser(refreshTokenResponse.data);
                    return instance(originalRequest);
                } else {
                    userStore.clearUser();
                    let signInUrl = '/sign-in';
                    const redirectUrl = window.location.search + window.location.hash
                    if (redirectUrl !== "" && redirectUrl !== "/") {
                        signInUrl += '?redirectUrl=' + encodeURIComponent(redirectUrl);
                    }
                    window.location.href = signInUrl
                    return Promise.reject(error);
                }
            } else {
                return Promise.reject(error);
            }
        } else if (error.request) {
            return Promise.reject(error);
        } else {
            return Promise.reject(error);
        }
    }
);

export default instance;
