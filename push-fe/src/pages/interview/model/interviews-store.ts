import { create } from 'zustand'
import type { InterviewSession } from '@/shared/api'
import { createInterview, listInterviews } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type InterviewsState = {
  items: InterviewSession[]
  status: LoadStatus
  error: string | null
  load: () => Promise<void>
  create: (body: {
    applicationId: string
    title: string
    scheduledAt: string
    notes?: string
  }) => Promise<InterviewSession>
}

export const useInterviewsStore = create<InterviewsState>()((set) => ({
  items: [],
  status: 'idle',
  error: null,
  load: async () => {
    set({ status: 'loading', error: null })
    try {
      const env = await listInterviews({ limit: 50 })
      set({ items: env.data, status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  create: async (body) => {
    const created = await createInterview(body)
    set((s) => ({ items: [created, ...s.items] }))
    return created
  },
}))
