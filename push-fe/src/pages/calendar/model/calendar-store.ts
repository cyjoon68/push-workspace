import { create } from 'zustand'
import type { CalendarEvent } from '@/shared/api'
import { listCalendarEvents, syncGoogle } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

const monthRange = (base: Date): { from: string; to: string } => {
  const from = new Date(base.getFullYear(), base.getMonth(), 1)
  const to = new Date(base.getFullYear(), base.getMonth() + 1, 0, 23, 59, 59)
  return { from: from.toISOString(), to: to.toISOString() }
}

type CalendarState = {
  items: CalendarEvent[]
  status: LoadStatus
  error: string | null
  syncing: boolean
  load: () => Promise<void>
  sync: () => Promise<void>
}

export const useCalendarStore = create<CalendarState>()((set) => ({
  items: [],
  status: 'idle',
  error: null,
  syncing: false,
  load: async () => {
    set({ status: 'loading', error: null })
    try {
      const env = await listCalendarEvents(monthRange(new Date()))
      set({ items: env.data, status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  sync: async () => {
    set({ syncing: true })
    try {
      await syncGoogle()
    } finally {
      set({ syncing: false })
    }
  },
}))
