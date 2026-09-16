import { useEffect } from 'react'
import { dDayLabel, formatDate, monthLabel } from '@/shared/lib/format'
import {
  Button,
  CanvasHeader,
  Card,
  DataList,
  DataListRow,
  EmptyState,
  ErrorState,
  Icon,
  Skeleton,
  StatusChip,
  showToast,
} from '@/shared/ui'
import { useCalendarStore } from '../model/calendar-store'

const TYPE_LABELS: Record<string, string> = {
  INTERVIEW: '면접',
  DEADLINE: '마감',
  FOLLOW_UP: '후속',
  CUSTOM: '일정',
}

const CalendarPage = () => {
  const items = useCalendarStore((s) => s.items)
  const status = useCalendarStore((s) => s.status)
  const error = useCalendarStore((s) => s.error)
  const syncing = useCalendarStore((s) => s.syncing)
  const load = useCalendarStore((s) => s.load)
  const sync = useCalendarStore((s) => s.sync)

  useEffect(() => {
    void load()
  }, [load])

  const onSync = async () => {
    try {
      await sync()
      showToast('동기화를 시작했어요', 'check')
      void load()
    } catch (e) {
      showToast(
        e instanceof Error ? e.message : '동기화하지 못했어요',
        'circle-alert',
      )
    }
  }

  const upcoming = [...items]
    .filter((e) => new Date(e.endsAt).getTime() >= Date.now() - 86_400_000)
    .sort((a, b) => a.startsAt.localeCompare(b.startsAt))

  const monthSummary = upcoming
    .slice(0, 5)
    .map((e) => `${formatDate(e.startsAt)} ${e.title}`)
    .join(' · ')

  return (
    <>
      <CanvasHeader
        title="캘린더"
        actions={
          <Button
            variant="primary"
            size="sm"
            loading={syncing}
            onClick={() => void onSync()}
          >
            <Icon name="calendar" size={20} /> 동기화
          </Button>
        }
      />
      <div className="canvas-body">
        {status === 'loading' ? (
          <>
            <Skeleton height={96} />
            <Skeleton height={52} />
            <Skeleton height={52} />
          </>
        ) : null}
        {status === 'error' ? (
          <ErrorState message={error ?? undefined} onRetry={() => void load()} />
        ) : null}
        {status === 'success' && items.length === 0 ? (
          <EmptyState message="다가오는 일정이 없어요" />
        ) : null}
        {status === 'success' && items.length > 0 ? (
          <>
            <Card title={monthLabel(new Date())}>
              <p className="t-body-sm">{monthSummary}</p>
            </Card>
            <DataList>
              {upcoming.map((event) => {
                const dday = dDayLabel(event.startsAt)
                return (
                  <DataListRow
                    key={event.id}
                    title={event.title}
                    meta={`${formatDate(event.startsAt)} · ${TYPE_LABELS[event.type]}`}
                    trailing={
                      <StatusChip
                        tone={dday === 'D-DAY' ? 'error' : 'ready'}
                        label={dday}
                      />
                    }
                  />
                )
              })}
            </DataList>
          </>
        ) : null}
      </div>
    </>
  )
}

export default CalendarPage
