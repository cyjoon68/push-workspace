import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Application } from '@/shared/api'

vi.mock('@/shared/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/shared/api')>()
  return {
    ...mod,
    listApplications: vi.fn(),
    createApplication: vi.fn(),
    patchApplication: vi.fn(),
  }
})

import {
  createApplication,
  listApplications,
  patchApplication,
} from '@/shared/api'
import { useApplicationsStore } from './applications-store'

const app = (over: Partial<Application> = {}): Application => ({
  id: 'a1',
  revision: 1,
  jobId: 'j1',
  company: '삼성전자',
  title: '백엔드',
  stage: 'DISCOVERED',
  notes: '',
  appliedAt: null,
  nextActionAt: null,
  createdAt: '2026-09-01T00:00:00Z',
  updatedAt: '2026-09-01T00:00:00Z',
  ...over,
})

describe('applications store', () => {
  beforeEach(() => {
    vi.mocked(listApplications).mockReset()
    vi.mocked(createApplication).mockReset()
    vi.mocked(patchApplication).mockReset()
    useApplicationsStore.setState({ items: [], status: 'idle', error: null })
  })

  it('transitions idle → loading → success with items', async () => {
    vi.mocked(listApplications).mockResolvedValue({
      data: [app()],
      page: { nextCursor: null, hasMore: false },
    })
    const pending = useApplicationsStore.getState().load()
    expect(useApplicationsStore.getState().status).toBe('loading')
    await pending
    const s = useApplicationsStore.getState()
    expect(s.status).toBe('success')
    expect(s.items).toHaveLength(1)
  })

  it('transitions to error and keeps the message', async () => {
    vi.mocked(listApplications).mockRejectedValue(new Error('boom'))
    await useApplicationsStore.getState().load()
    const s = useApplicationsStore.getState()
    expect(s.status).toBe('error')
    expect(s.error).toBe('boom')
  })

  it('moves stage with expectedRevision when transition is legal', async () => {
    const item = app({ stage: 'DISCOVERED', revision: 3 })
    useApplicationsStore.setState({ items: [item], status: 'success' })
    vi.mocked(patchApplication).mockResolvedValue(
      app({ stage: 'PREPARING', revision: 4 }),
    )
    await useApplicationsStore.getState().moveStage('a1', 'PREPARING')
    expect(patchApplication).toHaveBeenCalledWith('a1', {
      expectedRevision: 3,
      stage: 'PREPARING',
    })
    expect(useApplicationsStore.getState().items[0].stage).toBe('PREPARING')
  })

  it('does not call the API on an illegal transition', async () => {
    useApplicationsStore.setState({
      items: [app({ stage: 'ACCEPTED' })],
      status: 'success',
    })
    await useApplicationsStore.getState().moveStage('a1', 'DISCOVERED')
    expect(patchApplication).not.toHaveBeenCalled()
  })
})
