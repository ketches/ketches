import client from "./client";

export interface SignInResponse {
    user: {
        id: string;
        username: string;
        email: string;
        fullname: string;
        bio?: string;
        role: string;
    };
    must_change_password: boolean;
    default_password_notice: string;
}

export interface SignUpRequest {
    fullname: string;
    username: string;
    email: string;
    password: string;
    verification_code: string;
}

export interface SignUpConfigResponse {
    enabled: boolean;
}

export interface SignUpVerificationCodeResponse {
    expires_in_seconds: number;
    resend_after_seconds: number;
}

export const authApi = {
    signIn: async (data: { username: string; password: string }) => {
        return client.post(
            "/v1/users/sign-in",
            data,
        ) as Promise<SignInResponse>;
    },
    signUp: async (data: SignUpRequest) => {
        return client.post("/v1/users/sign-up", data);
    },
    getSignUpConfig: async () => {
        return client.get("/v1/users/sign-up/config") as Promise<SignUpConfigResponse>;
    },
    sendSignUpVerificationCode: async (data: { email: string }) => {
        return client.post("/v1/users/sign-up/verification-code", data) as Promise<SignUpVerificationCodeResponse>;
    },
    logout: async () => {
        return client.post("/v1/users/logout", {}) as Promise<void>;
    }
};
