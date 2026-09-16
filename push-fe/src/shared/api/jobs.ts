import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type JobPosting = {
  id: string
  revision: number
  company: string
  title: string
  sourceKind: 'URL' | 'TEXT' | 'DOM'
  sourceUrl: string | null
  sourceText: string
  requirements: string[]
  preferred: string[]
  keywords: string[]
  risks: string[]
  deadline: string | null
  language: string | null
  createdAt: string
  updatedAt: string
}

export type GapAnalysis = {
  id: string
  applicationId: string
  jobId: string
  jobRevision: number
  evidenceIds: string[]
  matched: { requirement: string; evidenceIds: string[] }[]
  missing: string[]
  preferredMissing: string[]
  risks: string[]
  fitScore: number | null
  method: 'RULE_BASED' | 'AI_ASSISTED'
  createdAt: string
}

export const listJobs = async (
  params: ListParams & { query?: string; archived?: boolean } = {},
): Promise<ListEnvelope<JobPosting>> =>
  request(() =>
    api.get('jobs', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.query ? { query: params.query } : {}),
        ...(params.archived !== undefined
          ? { archived: String(params.archived) }
          : {}),
      },
    }),
  )

export const getJob = async (id: string): Promise<JobPosting> => {
  const env = await request<DataEnvelope<JobPosting>>(() =>
    api.get(`jobs/${id}`),
  )
  return env.data
}

export const createJob = async (body: {
  company: string
  title: string
  sourceKind: 'URL' | 'TEXT' | 'DOM'
  sourceUrl?: string
  sourceText: string
  requirements?: string[]
  preferred?: string[]
  deadline?: string
  language?: string
}): Promise<JobPosting> => {
  const env = await request<DataEnvelope<JobPosting>>(() =>
    api.post('jobs', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const listJobAnalyses = async (
  jobId: string,
  params: ListParams = {},
): Promise<ListEnvelope<GapAnalysis>> =>
  request(() =>
    api.get(`jobs/${jobId}/analyses`, {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
      },
    }),
  )
