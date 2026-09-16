import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'
import type { AiOptions, Operation } from './conversations'

export type InterviewSession = {
  id: string
  revision: number
  applicationId: string
  title: string
  scheduledAt: string
  durationMinutes: number | null
  eventId: string | null
  evidenceIds: string[]
  notes: string
  reflection: string
  createdAt: string
  updatedAt: string
}

export const listInterviews = async (
  params: ListParams & { applicationId?: string } = {},
): Promise<ListEnvelope<InterviewSession>> =>
  request(() =>
    api.get('interviews', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
      },
    }),
  )

export const createInterview = async (body: {
  applicationId: string
  title: string
  scheduledAt: string
  durationMinutes?: number
  evidenceIds?: string[]
  notes?: string
}): Promise<InterviewSession> => {
  const env = await request<DataEnvelope<InterviewSession>>(() =>
    api.post('interviews', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const prepareInterview = async (
  id: string,
  body: { expectedRevision: number; ai: AiOptions | null },
): Promise<Operation> => {
  const env = await request<DataEnvelope<Operation>>(() =>
    api.post(`interviews/${id}/prepare`, {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}
