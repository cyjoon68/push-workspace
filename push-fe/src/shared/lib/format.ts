const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

export const formatRelativeTime = (iso: string): string => {
  const diff = Date.now() - new Date(iso).getTime()
  if (Number.isNaN(diff) || diff < 0) return ''
  if (diff < HOUR_MS) return `${Math.max(1, Math.floor(diff / MINUTE_MS))}분 전`
  if (diff < DAY_MS) return `${Math.floor(diff / HOUR_MS)}시간 전`
  const days = Math.floor(diff / DAY_MS)
  if (days === 1) return '어제'
  if (days < 30) return `${days}일 전`
  return new Date(iso).toLocaleDateString('ko-KR')
}

export const formatDate = (iso: string): string =>
  new Date(iso).toLocaleDateString('ko-KR', { month: 'long', day: 'numeric' })

export const formatDateTime = (iso: string): string =>
  new Date(iso).toLocaleString('ko-KR', {
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

export const dDayLabel = (isoDate: string): string => {
  const target = new Date(isoDate)
  const today = new Date()
  const startOfDay = (d: Date) =>
    new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const diffDays = Math.round(
    (startOfDay(target) - startOfDay(today)) / DAY_MS,
  )
  if (diffDays === 0) return 'D-DAY'
  if (diffDays > 0) return `D-${diffDays}`
  return `D+${Math.abs(diffDays)}`
}

export const monthLabel = (date: Date): string =>
  `${date.getFullYear()}년 ${date.getMonth() + 1}월`
