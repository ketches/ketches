import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
// If useRouter() causes issues at the top level, import the router instance directly:
// import router from '@/router'; // Assuming your router is configured and exported
import { useUserStore } from '@/stores/userStore'; // Adjust the import path as necessary
import { useRouter } from 'vue-router';
import { toast } from 'vue-sonner';


const apibaseURL = import.meta.env.VITE_API_BASE_URL;

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
        // This part usually handles errors in setting up the request, not server responses.
        toast.error("请求错误", {
            description: error.message || '请求配置错误，请稍后再试。',
        });
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
                    const routerInstance = useRouter();
                    routerInstance.push('/sign-in?redirectUrl=' + encodeURIComponent(routerInstance.currentRoute.value.fullPath));
                    return Promise.reject(error);
                }

                // Retry to refresh token
                const refreshTokenResponse = await instance.post('/users/refresh-token');
                if (refreshTokenResponse.data.userID) { // Check custom success code
                    userStore.setUser(refreshTokenResponse.data);
                    return instance(originalRequest);
                } else {
                    userStore.clearUser();
                    const routerInstance = useRouter();
                    routerInstance.push('/sign-in?redirectUrl=' + encodeURIComponent(routerInstance.currentRoute.value.fullPath));
                    return Promise.reject(error);
                }
            } else {
                // For non-401 errors, display the error from response body
                const errorMessage = (error.response?.data as any)?.error || '请求失败，请稍后再试。';
                toast.dismiss();
                toast.error(error.response?.statusText || "请求错误", {
                    description: errorMessage,
                });
                return Promise.reject(error);
            }
        } else if (error.request) {
            toast.dismiss();
            toast.error("网络错误", {
                description: '请求未响应，请检查网络连接或稍后再试。',
            });
            return Promise.reject(error);
        } else {
            toast.dismiss();
            toast.error("请求错误", {
                description: error.message || '请求配置错误，请稍后再试。',
            });
            return Promise.reject(error);
        }
    }
);

export default instance;
