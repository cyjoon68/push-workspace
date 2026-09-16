import { create } from 'zustand'
import type { GapAnalysis, JobPosting } from '@/shared/api'
import { getJob, listJobAnalyses } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type JobState = {
  current: JobPosting | null
  analyses: GapAnalysis[]
  status: LoadStatus
  error: string | null
  load: (id: string) => Promise<void>
  reset: () => void
}

export const useJobStore = create<JobState>()((set) => ({
  current: null,
  analyses: [],
  status: 'idle',
  error: null,
  load: async (id) => {
    set({ status: 'loading', error: null })
    try {
      const job = await getJob(id)
      let analyses: GapAnalysis[] = []
      try {
        const env = await listJobAnalyses(id, { limit: 10 })
        analyses = env.data
      } catch {
        analyses = []
      }
      set({ current: job, analyses, status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  reset: () =>
    set({ current: null, analyses: [], status: 'idle', error: null }),
}))
