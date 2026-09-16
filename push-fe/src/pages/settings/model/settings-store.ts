import { create } from 'zustand'
import type { AiModel, AuthUser, BillingSummary } from '@/shared/api'
import { fetchBilling, fetchMe, listAiModels } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

const APPEARANCE_KEY = 'push-appearance'

type SettingsState = {
  me: AuthUser | null
  billing: BillingSummary | null
  models: AiModel[]
  status: LoadStatus
  error: string | null
  theme: string
  setTheme: (theme: string) => void
  load: () => Promise<void>
}

export const useSettingsStore = create<SettingsState>()((set) => ({
  me: null,
  billing: null,
  models: [],
  status: 'idle',
  error: null,
  theme: localStorage.getItem(APPEARANCE_KEY) ?? '라이트',
  setTheme: (theme) => set({ theme }),
  load: async () => {
    set({ status: 'loading', error: null })
    try {
      const [me, billing, models] = await Promise.allSettled([
        fetchMe(),
        fetchBilling(),
        listAiModels({ provider: 'OPENAI', credentialMode: 'MANAGED' }),
      ])
      set({
        me: me.status === 'fulfilled' ? me.value : null,
        billing: billing.status === 'fulfilled' ? billing.value : null,
        models:
          models.status === 'fulfilled' ? models.value.data : [],
        status: 'success',
      })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
}))

export const persistTheme = (theme: string): void => {
  localStorage.setItem(APPEARANCE_KEY, theme)
}
