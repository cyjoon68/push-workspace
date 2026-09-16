import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type Conversation = {
  id: string
  revision: number
  applicationId: string | null
  projectId?: string
  title: string
  pinned: boolean
  createdAt: string
  updatedAt: string
}

export type MessageRole = 'USER' | 'ASSISTANT' | 'SYSTEM'

export type MessageAttachment =
  | { type: 'DOCUMENT_VERSION'; id: string; documentId: string; title: string }
  | { type: 'EVIDENCE'; id: string; title: string }
  | { type: 'APPROVAL'; id: string }

export type Message = {
  id: string
  conversationId: string
  role: MessageRole
  text: string
  attachments: MessageAttachment[]
  operationId: string | null
  createdAt: string
}

export type AiOptions = {
  provider: 'OPENAI' | 'CLAUDE' | 'GEMINI' | 'GROK'
  model: string
  credentialMode: 'MANAGED' | 'BYOK'
  effort: 'LOW' | 'MEDIUM' | 'HIGH'
}

export type AccessMode = 'SUGGEST' | 'CONFIRM_ACTIONS'

export const listConversations = async (
  params: ListParams & { applicationId?: string } = {},
): Promise<ListEnvelope<Conversation>> =>
  request(() =>
    api.get('conversations', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
      },
    }),
  )

export const createConversation = async (body: {
  applicationId: string | null
  title?: string
}): Promise<Conversation> => {
  const env = await request<DataEnvelope<Conversation>>(() =>
    api.post('conversations', {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const listMessages = async (
  conversationId: string,
  params: ListParams = {},
): Promise<ListEnvelope<Message>> =>
  request(() =>
    api.get(`conversations/${conversationId}/messages`, {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
      },
    }),
  )

export const sendMessage = async (
  conversationId: string,
  body: {
    text: string
    context: { documentId?: string; versionId?: string; evidenceIds: string[] }
    ai: AiOptions
    accessMode: AccessMode
  },
): Promise<Operation> => {
  const env = await request<DataEnvelope<Operation>>(() =>
    api.post(`conversations/${conversationId}/messages`, {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export type OperationStatus =
  | 'QUEUED'
  | 'RUNNING'
  | 'NEEDS_INPUT'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'CANCELLED'

export type Operation = {
  id: string
  type: string
  applicationId: string | null
  status: OperationStatus
  progress: number | null
  result: unknown
  error: { code: string; message: string; retryable: boolean } | null
  inputRequest: {
    code: string
    message: string
    fields: { name: string; label: string; type: 'TEXT' | 'FILE' }[]
  } | null
  createdAt: string
  updatedAt: string
}
