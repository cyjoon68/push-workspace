import { api } from './client'
import { request } from './envelope'
import type { ListEnvelope, ListParams } from './envelope'
import type { Operation } from './conversations'

export type CalendarEvent = {
  id: string
  revision: number
  applicationId: string | null
  type: 'INTERVIEW' | 'DEADLINE' | 'FOLLOW_UP' | 'CUSTOM'
  title: string
  startsAt: string
  endsAt: string
  timeZone: string
  source: 'LOCAL' | 'GOOGLE'
  externalId: string | null
  notes: string
  createdAt: string
  updatedAt: string
}

export const listCalendarEvents = async (
  params: ListParams & { from: string; to: string; applicationId?: string },
): Promise<ListEnvelope<CalendarEvent>> =>
  request(() =>
    api.get('calendar/events', {
      searchParams: {
        from: params.from,
        to: params.to,
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
      },
    }),
  )

export const syncGoogle = async (): Promise<Operation> => {
  const env = await request<{ data: Operation }>(() =>
    api.post('integrations/google/sync', { json: {} }),
  )
  return env.data
}
