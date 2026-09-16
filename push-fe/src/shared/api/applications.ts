import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type ApplicationStage =
  | 'DISCOVERED'
  | 'PREPARING'
  | 'READY'
  | 'APPLIED'
  | 'SCREENING'
  | 'INTERVIEW'
  | 'OFFER'
  | 'ACCEPTED'
  | 'REJECTED'
  | 'WITHDRAWN'

export type Application = {
  id: string
  revision: number
  jobId: string
  company: string
  title: string
  stage: ApplicationStage
  notes: string
  appliedAt: string | null
  nextActionAt: string | null
  createdAt: string
  updatedAt: string
}

export const listApplications = async (
  params: ListParams & { stage?: ApplicationStage; query?: string } = {},
): Promise<ListEnvelope<Application>> =>
  request(() =>
    api.get('applications', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.stage ? { stage: params.stage } : {}),
        ...(params.query ? { query: params.query } : {}),
      },
    }),
  )

export const getApplication = async (id: string): Promise<Application> => {
  const env = await request<DataEnvelope<Application>>(() =>
    api.get(`applications/${id}`),
  )
  return env.data
}

export const createApplication = async (body: {
  jobId: string
  notes?: string
}): Promise<Application> => {
  const env = await request<DataEnvelope<Application>>(() =>
    api.post('applications', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const patchApplication = async (
  id: string,
  body: {
    expectedRevision: number
    stage?: ApplicationStage
    notes?: string
    nextActionAt?: string
  },
): Promise<Application> => {
  const env = await request<DataEnvelope<Application>>(() =>
    api.patch(`applications/${id}`, { json: body }),
  )
  return env.data
}
