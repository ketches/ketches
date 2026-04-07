import client from "./client"
import type { OperationRequestBody, OperationResponseData, WithRequired } from "./generated/helpers"

export type SignInRequest = OperationRequestBody<"/api/v1/users/sign-in", "post">
type GeneratedSignInResponse = OperationResponseData<"/api/v1/users/sign-in", "post">
type GeneratedUser = NonNullable<GeneratedSignInResponse["user"]>

export type SignInResponse = WithRequired<
  GeneratedSignInResponse,
  "user" | "must_change_password" | "default_password_notice"
> & {
  user: WithRequired<GeneratedUser, "id" | "username" | "email" | "fullname" | "role">
}

export type SignUpRequest = OperationRequestBody<"/api/v1/users/sign-up", "post">
export type SignUpResponse = OperationResponseData<"/api/v1/users/sign-up", "post", 201>

export type SignUpConfigResponse = WithRequired<
  OperationResponseData<"/api/v1/users/sign-up/config", "get">,
  "enabled"
>

export type SignUpVerificationCodeRequest = OperationRequestBody<
  "/api/v1/users/sign-up/verification-code",
  "post"
>
export type SignUpVerificationCodeResponse = WithRequired<
  OperationResponseData<"/api/v1/users/sign-up/verification-code", "post">,
  "expires_in_seconds" | "resend_after_seconds"
>

export const authApi = {
  signIn: async (data: SignInRequest) => {
    return client.post("/v1/users/sign-in", data) as Promise<SignInResponse>
  },
  signUp: async (data: SignUpRequest) => {
    return client.post("/v1/users/sign-up", data) as Promise<SignUpResponse>
  },
  getSignUpConfig: async () => {
    return client.get("/v1/users/sign-up/config") as Promise<SignUpConfigResponse>
  },
  sendSignUpVerificationCode: async (data: SignUpVerificationCodeRequest) => {
    return client.post("/v1/users/sign-up/verification-code", data) as Promise<SignUpVerificationCodeResponse>
  },
  logout: async () => {
    return client.post("/v1/users/logout", {}) as Promise<void>
  },
}
