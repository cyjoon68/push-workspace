import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'

export type ApprovalKind =
  | 'EVIDENCE_USE'
  | 'DOCUMENT_FINALIZE'
  | 'APPLICATION_SUBMIT'
  | 'CLI_EXECUTE'

export type ApprovalStatus =
  | 'PENDING'
  | 'APPROVED'
  | 'DENIED'
  | 'EXPIRED'
  | 'CONSUMED'

export type Approval = {
  id: string
  revision: number
  kind: ApprovalKind
  applicationId: string
  targetId: string
  targetRevision: number | null
  payloadHash: string
  status: ApprovalStatus
  expiresAt: string | null
  decidedAt: string | null
  consumedAt: string | null
  createdAt: string
  updatedAt: string
}

export type ApprovalSummary = {
  approval: Approval
  targetSummary: {
    title?: string
    workingDirectory?: string
    executable?: string
    arguments?: string[]
    prompt?: string
  } | null
}

export const getApproval = async (id: string): Promise<ApprovalSummary> =>
  request(() => api.get(`approvals/${id}`))

export const listApprovals = async (
  params: ListParams & { applicationId?: string; status?: ApprovalStatus } = {},
): Promise<ListEnvelope<Approval>> =>
  request(() =>
    api.get('approvals', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
        ...(params.status ? { status: params.status } : {}),
      },
    }),
  )

export const decideApproval = async (
  id: string,
  body: { expectedRevision: number; decision: 'APPROVED' | 'DENIED' },
): Promise<Approval> => {
  const env = await request<DataEnvelope<Approval>>(() =>
    api.post(`approvals/${id}/decision`, { json: body }),
  )
  return env.data
}
