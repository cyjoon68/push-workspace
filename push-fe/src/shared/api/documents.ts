import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type DocumentKind = 'RESUME' | 'PORTFOLIO' | 'COVER_LETTER'
export type DocumentTemplate = 'CLASSIC' | 'MODERN' | 'COMPACT'
export type DocumentStatus = 'DRAFT' | 'FINALIZED' | 'ARCHIVED'

export type PushDocument = {
  id: string
  revision: number
  applicationId: string
  title: string
  kind: DocumentKind
  template: DocumentTemplate
  language: string | null
  status: DocumentStatus
  latestVersionId: string | null
  finalizedVersionId: string | null
  createdAt: string
  updatedAt: string
}

export type EvidenceRef = {
  evidenceId: string
  start: number
  end: number
}

export type ClaimStatus = 'SUPPORTED' | 'NEEDS_REVIEW' | 'UNSUPPORTED'

export type DocumentBlock = {
  id: string
  text: string
  evidenceRefs: EvidenceRef[]
  claimStatus: ClaimStatus
}

export type DocumentVersion = {
  id: string
  documentId: string
  applicationId: string
  number: number
  content: Record<string, unknown>
  blocks: DocumentBlock[]
  changeNote: string | null
  createdAt: string
}

export type DocumentExport = {
  id: string
  documentId: string
  versionId: string
  format: 'PDF' | 'DOCX'
  template: DocumentTemplate
  language: string | null
  status: 'READY_TO_RENDER'
}

export const listDocuments = async (
  params: ListParams & { applicationId?: string; kind?: DocumentKind } = {},
): Promise<ListEnvelope<PushDocument>> =>
  request(() =>
    api.get('documents', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
        ...(params.kind ? { kind: params.kind } : {}),
      },
    }),
  )

export const getDocument = async (id: string): Promise<PushDocument> => {
  const env = await request<DataEnvelope<PushDocument>>(() =>
    api.get(`documents/${id}`),
  )
  return env.data
}

export const createDocument = async (body: {
  applicationId: string
  title: string
  kind: DocumentKind
  template: DocumentTemplate
  language?: string
}): Promise<PushDocument> => {
  const env = await request<DataEnvelope<PushDocument>>(() =>
    api.post('documents', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const listDocumentVersions = async (
  documentId: string,
  params: ListParams = {},
): Promise<ListEnvelope<DocumentVersion>> =>
  request(() =>
    api.get(`documents/${documentId}/versions`, {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
      },
    }),
  )

export const getDocumentVersion = async (
  documentId: string,
  versionId: string,
): Promise<DocumentVersion> => {
  const env = await request<DataEnvelope<DocumentVersion>>(() =>
    api.get(`documents/${documentId}/versions/${versionId}`),
  )
  return env.data
}

export const createDocumentExport = async (
  documentId: string,
  body: {
    versionId: string
    format: 'PDF' | 'DOCX'
    rendererVersion: string
  },
): Promise<DocumentExport> => {
  const env = await request<DataEnvelope<DocumentExport>>(() =>
    api.post(`documents/${documentId}/exports`, {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}
