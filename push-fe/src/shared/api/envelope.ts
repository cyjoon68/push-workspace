import { HTTPError, TimeoutError } from 'ky'
import type { ResponsePromise } from 'ky'

export type ApiErrorBody = {
  code: string
  message: string
  requestId?: string
  details?: Record<string, unknown>
}

export type PageInfo = {
  nextCursor: string | null
  hasMore: boolean
}

export type DataEnvelope<T> = { data: T }

export type ListEnvelope<T> = { data: T[]; page: PageInfo }

export type ListParams = {
  limit?: number
  cursor?: string
}

export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly details?: Record<string, unknown>

  constructor(
    message: string,
    code: string,
    status: number,
    details?: Record<string, unknown>,
  ) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.details = details
  }
}

export const NETWORK_ERROR_MESSAGE =
  '네트워크 상태를 확인한 뒤 다시 시도해 주세요.'

export const toApiError = async (error: unknown): Promise<ApiError> => {
  if (error instanceof HTTPError) {
    try {
      const body = (await error.response.clone().json()) as {
        error?: ApiErrorBody
      }
      if (body.error) {
        return new ApiError(
          body.error.message,
          body.error.code,
          error.response.status,
          body.error.details,
        )
      }
    } catch {
      return new ApiError(error.message, 'HTTP_ERROR', error.response.status)
    }
    return new ApiError(error.message, 'HTTP_ERROR', error.response.status)
  }
  if (error instanceof TimeoutError) {
    return new ApiError(NETWORK_ERROR_MESSAGE, 'TIMEOUT', 0)
  }
  if (error instanceof ApiError) return error
  return new ApiError(NETWORK_ERROR_MESSAGE, 'NETWORK_ERROR', 0)
}

export const request = async <T>(call: () => ResponsePromise): Promise<T> => {
  try {
    const response = await call()
    if (response.status === 204) return undefined as T
    return (await response.json()) as T
  } catch (error) {
    throw await toApiError(error)
  }
}
