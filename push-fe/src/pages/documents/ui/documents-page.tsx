import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/shared/constants'
import { formatRelativeTime } from '@/shared/lib/format'
import type { DocumentKind } from '@/shared/api'
import {
  CanvasHeader,
  Button,
  CardGrid,
  DocCard,
  EmptyState,
  ErrorState,
  Icon,
  SkeletonCardGrid,
} from '@/shared/ui'
import { useDocumentsStore } from '@/entities/document'
import NewDocumentDialog from './new-document-dialog'

const KIND_LABELS: Record<DocumentKind, string> = {
  RESUME: '이력서',
  COVER_LETTER: '자기소개서',
  PORTFOLIO: '포트폴리오',
}

const DocumentsPage = () => {
  const navigate = useNavigate()
  const items = useDocumentsStore((s) => s.items)
  const status = useDocumentsStore((s) => s.status)
  const error = useDocumentsStore((s) => s.error)
  const load = useDocumentsStore((s) => s.load)
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    void load()
  }, [load])

  return (
    <>
      <CanvasHeader
        title="내 서류"
        actions={
          <Button
            variant="primary"
            size="sm"
            onClick={() => setDialogOpen(true)}
          >
            <Icon name="plus" size={20} /> 새 문서
          </Button>
        }
      />
      <div className="canvas-body">
        {status === 'loading' ? <SkeletonCardGrid count={3} /> : null}
        {status === 'error' ? (
          <ErrorState message={error ?? undefined} onRetry={() => void load()} />
        ) : null}
        {status === 'success' && items.length === 0 ? (
          <EmptyState
            message="아직 문서가 없어요"
            actionLabel="새 문서"
            onAction={() => setDialogOpen(true)}
          />
        ) : null}
        {status === 'success' && items.length > 0 ? (
          <CardGrid>
            {items.map((doc) => (
              <DocCard
                key={doc.id}
                title={doc.title}
                meta={`${KIND_LABELS[doc.kind]} · ${formatRelativeTime(doc.updatedAt)}`}
                onOpen={() => navigate(ROUTES.document(doc.id))}
              />
            ))}
          </CardGrid>
        ) : null}
      </div>
      <NewDocumentDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    </>
  )
}

export default DocumentsPage
