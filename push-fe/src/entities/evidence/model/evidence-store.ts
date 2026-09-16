import { create } from 'zustand'
import type { CareerEvidence } from '@/shared/api'
import { createCareerEvidence, listCareerEvidence } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type EvidenceState = {
  items: CareerEvidence[]
  status: LoadStatus
  error: string | null
  load: () => Promise<void>
  create: (body: {
    kind: CareerEvidence['kind']
    title: string
    sourceText: string
    sourceUrl?: string
    skills?: string[]
  }) => Promise<CareerEvidence>
}

export const useEvidenceStore = create<EvidenceState>()((set) => ({
  items: [],
  status: 'idle',
  error: null,
  load: async () => {
    set({ status: 'loading', error: null })
    try {
      const env = await listCareerEvidence({ limit: 50 })
      set({ items: env.data, status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  create: async (body) => {
    const created = await createCareerEvidence(body)
    set((s) => ({ items: [created, ...s.items] }))
    return created
  },
}))
