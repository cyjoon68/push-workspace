import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope } from './envelope'

export type AiModel = {
  provider: 'OPENAI' | 'CLAUDE' | 'GEMINI' | 'GROK'
  model: string
  label: string
  available: boolean
  supportedEfforts: ('LOW' | 'MEDIUM' | 'HIGH')[]
}

export const listAiModels = async (params?: {
  provider?: string
  credentialMode?: 'MANAGED' | 'BYOK'
}): Promise<ListEnvelope<AiModel>> =>
  request(() =>
    api.get('ai/models', {
      searchParams: {
        ...(params?.provider ? { provider: params.provider } : {}),
        ...(params?.credentialMode
          ? { credentialMode: params.credentialMode }
          : {}),
      },
    }),
  )

export type BillingSummary = {
  subscriptionStatus: string
  plan: string
  periodEndsAt: string | null
  balanceMicroCredits: number
  reservedMicroCredits: number
}

export const fetchBilling = async (): Promise<BillingSummary> => {
  const env = await request<DataEnvelope<BillingSummary>>(() =>
    api.get('billing'),
  )
  return env.data
}
