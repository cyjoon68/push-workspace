import { create } from 'zustand'
import type { Application, ApplicationStage } from '@/shared/api'
import {
  createApplication,
  listApplications,
  patchApplication,
} from '@/shared/api'
import { canTransition } from './application-stage'

export type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type ApplicationsState = {
  items: Application[]
  status: LoadStatus
  error: string | null
  load: () => Promise<void>
  create: (jobId: string) => Promise<Application>
  moveStage: (id: string, to: ApplicationStage) => Promise<void>
}

export const useApplicationsStore = create<ApplicationsState>()(
  (set, get) => ({
    items: [],
    status: 'idle',
    error: null,
    load: async () => {
      set({ status: 'loading', error: null })
      try {
        const env = await listApplications({ limit: 50 })
        set({ items: env.data, status: 'success' })
      } catch (error) {
        set({
          status: 'error',
          error: error instanceof Error ? error.message : 'error',
        })
      }
    },
    create: async (jobId) => {
      const created = await createApplication({ jobId })
      set((s) => ({ items: [created, ...s.items] }))
      return created
    },
    moveStage: async (id, to) => {
      const target = get().items.find((a) => a.id === id)
      if (!target) return
      if (!canTransition(target.stage, to)) return
      const updated = await patchApplication(id, {
        expectedRevision: target.revision,
        stage: to,
      })
      set((s) => ({
        items: s.items.map((a) => (a.id === id ? updated : a)),
      }))
    },
  }),
)
