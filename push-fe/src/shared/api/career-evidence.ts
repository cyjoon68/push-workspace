import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type EvidenceKind =
  | 'RESUME'
  | 'GITHUB'
  | 'CAREER'
  | 'EDUCATION'
  | 'SKILL'
  | 'PROJECT'

export type VerificationStatus =
  | 'USER_PROVIDED'
  | 'PENDING'
  | 'VERIFIED'
  | 'REJECTED'

export type CareerEvidence = {
  id: string
  revision: number
  createdAt: string
  updatedAt: string
  kind: EvidenceKind
  title: string
  sourceText: string
  sourceUrl: string | null
  skills: string[]
  verificationStatus: VerificationStatus
  provenance: {
    sourceId: string | null
    projectEvidenceId: string | null
    contentHash: string
    sourceLocation: {
      start: number
      end: number
      unit: 'CODE_POINT'
    } | null
  }
}

export const listCareerEvidence = async (
  params: ListParams & { kind?: EvidenceKind; query?: string } = {},
): Promise<ListEnvelope<CareerEvidence>> =>
  request(() =>
    api.get('career-evidence', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.kind ? { kind: params.kind } : {}),
        ...(params.query ? { query: params.query } : {}),
      },
    }),
  )

export const createCareerEvidence = async (body: {
  kind: EvidenceKind
  title: string
  sourceText: string
  sourceUrl?: string
  skills?: string[]
  supersedesId?: string
}): Promise<CareerEvidence> => {
  const env = await request<DataEnvelope<CareerEvidence>>(() =>
    api.post('career-evidence', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}
