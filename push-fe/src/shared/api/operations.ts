import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope } from './envelope'
import type { Operation } from './conversations'

export const getOperation = async (id: string): Promise<Operation> => {
  const env = await request<DataEnvelope<Operation>>(() =>
    api.get(`operations/${id}`),
  )
  return env.data
}

const TERMINAL: Operation['status'][] = [
  'SUCCEEDED',
  'FAILED',
  'CANCELLED',
  'NEEDS_INPUT',
]

const sleep = (ms: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, ms))

export const waitForOperation = async (
  id: string,
  options: { intervalMs?: number; timeoutMs?: number } = {},
): Promise<Operation> => {
  const intervalMs = options.intervalMs ?? 1_000
  const timeoutMs = options.timeoutMs ?? 60_000
  const deadline = Date.now() + timeoutMs
  for (;;) {
    const operation = await getOperation(id)
    if (TERMINAL.includes(operation.status)) return operation
    if (Date.now() >= deadline) return operation
    await sleep(intervalMs)
  }
}
