import { useEffect, useState } from 'react'
import { formatDateTime } from '@/shared/lib/format'
import {
  Button,
  CanvasHeader,
  Card,
  DataList,
  DataListRow,
  EmptyState,
  ErrorState,
  Icon,
  SkeletonRows,
} from '@/shared/ui'
import { useInterviewsStore } from '../model/interviews-store'
import InterviewAddDialog from './interview-add-dialog'

const InterviewPage = () => {
  const items = useInterviewsStore((s) => s.items)
  const status = useInterviewsStore((s) => s.status)
  const error = useInterviewsStore((s) => s.error)
  const load = useInterviewsStore((s) => s.load)
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    void load()
  }, [load])

  const reflected = items.filter((s) => s.reflection)

  return (
    <>
      <CanvasHeader
        title="면접"
        actions={
          <Button
            variant="primary"
            size="sm"
            onClick={() => setDialogOpen(true)}
          >
            <Icon name="plus" size={20} /> 면접 준비
          </Button>
        }
      />
      <div className="canvas-body">
        {status === 'loading' ? (
          <DataList>
            <SkeletonRows count={3} height={52} />
          </DataList>
        ) : null}
        {status === 'error' ? (
          <ErrorState message={error ?? undefined} onRetry={() => void load()} />
        ) : null}
        {status === 'success' && items.length === 0 ? (
          <EmptyState
            message="아직 면접 준비가 없어요"
            actionLabel="면접 준비"
            onAction={() => setDialogOpen(true)}
          />
        ) : null}
        {status === 'success' && items.length > 0 ? (
          <>
            <DataList>
              {items.map((session) => (
                <DataListRow
                  key={session.id}
                  title={session.title}
                  meta={`${formatDateTime(session.scheduledAt)}${session.durationMinutes ? ` · ${session.durationMinutes}분` : ''}`}
                />
              ))}
            </DataList>
            {reflected.map((session) => (
              <Card key={session.id} title={`회고 — ${session.title}`}>
                <p className="t-body-sm">{session.reflection}</p>
              </Card>
            ))}
          </>
        ) : null}
      </div>
      <InterviewAddDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    </>
  )
}

export default InterviewPage
