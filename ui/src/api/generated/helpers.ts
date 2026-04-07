import type { paths } from "./openapi"

type HttpMethod = "get" | "put" | "post" | "delete" | "patch"

type JsonContent<T> = T extends { "application/json": infer TJson } ? TJson : never

type OperationAt<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
> = NonNullable<paths[TPath][TMethod]>

type OperationResponses<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
> = OperationAt<TPath, TMethod> extends { responses: infer TResponses } ? TResponses : never

type DefaultSuccessStatus<TResponses> =
  200 extends keyof TResponses ? 200 :
  201 extends keyof TResponses ? 201 :
  204 extends keyof TResponses ? 204 :
  never

type ResponseJson<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
  TStatus extends keyof OperationResponses<TPath, TMethod>,
> = OperationResponses<TPath, TMethod>[TStatus] extends { content: infer TContent }
  ? JsonContent<TContent>
  : never

type ResponseData<TResponse> = TResponse extends { data: infer TData } ? TData : never

export type OperationRequestBody<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
> = OperationAt<TPath, TMethod> extends { requestBody: { content: infer TContent } }
  ? JsonContent<TContent>
  : never

export type OperationResponseData<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
  TStatus extends keyof OperationResponses<TPath, TMethod> = DefaultSuccessStatus<OperationResponses<TPath, TMethod>>,
> = ResponseData<ResponseJson<TPath, TMethod, TStatus>>

export type OperationQueryParameters<
  TPath extends keyof paths,
  TMethod extends HttpMethod,
> = OperationAt<TPath, TMethod> extends { parameters: { query?: infer TQuery } }
  ? NonNullable<TQuery>
  : never

export type WithRequired<T, TKey extends keyof T> = T & {
  [K in TKey]-?: NonNullable<T[K]>
}
