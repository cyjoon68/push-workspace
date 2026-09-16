import { useEffect, useState } from 'react'
import type { Approval, ApprovalKind } from '@/shared/api'
import { useApprovalsStore } from '@/entities/approval'
import { Button, Icon, StatusChip, showToast } from '@/shared/ui'

const KIND_TITLES: Record<ApprovalKind, string> = {
  CLI_EXECUTE: 'CLI 실행 승인',
  DOCUMENT_FINALIZE: '문서 확정',
  APPLICATION_SUBMIT: '지원 제출 승인',
  EVIDENCE_USE: '근거 사용 승인',
}

const STATUS_LABELS: Record<Approval['status'], string> = {
  PENDING: '대기 중',
  APPROVED: '승인됨',
  DENIED: '거부됨',
  EXPIRED: '만료됨',
  CONSUMED: '처리됨',
}

type ApprovalCardProps = {
  approvalId: string
}

const ApprovalCard = ({ approvalId }: ApprovalCardProps) => {
  const approval = useApprovalsStore((s) => s.byId[approvalId])
  const summary = useApprovalsStore((s) => s.summaries[approvalId])
  const ensure = useApprovalsStore((s) => s.ensure)
  const decide = useApprovalsStore((s) => s.decide)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    void ensure(approvalId)
  }, [approvalId, ensure])

  if (!approval) return null

  const onDecide = async (decision: 'APPROVED' | 'DENIED') => {
    setBusy(true)
    try {
      await decide(approvalId, decision)
      showToast(decision === 'APPROVED' ? '승인했어요' : '거부했어요')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '처리하지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  const title = summary?.title
    ? `${KIND_TITLES[approval.kind]} — ${summary.title}`
    : KIND_TITLES[approval.kind]

  const command =
    summary?.executable || summary?.prompt
      ? [
          summary.executable
            ? `$ ${summary.executable}${summary.arguments?.length ? ` ${summary.arguments.join(' ')}` : ''}`
            : null,
          summary.workingDirectory
            ? `디렉터리: ${summary.workingDirectory}`
            : null,
          summary.prompt ? `프롬프트: ${summary.prompt}` : null,
        ]
          .filter(Boolean)
          .join('\n')
      : null

  return (
    <div className="approval-card">
      <div className="approval-card-title">{title}</div>
      {command ? <div className="approval-card-cmd">{command}</div> : null}
      <div className="approval-card-body">
        {approval.status === 'PENDING'
          ? '내용을 확인한 뒤 승인하거나 거부할 수 있어요.'
          : STATUS_LABELS[approval.status]}
      </div>
      {approval.status === 'PENDING' ? (
        <div className="approval-card-actions">
          <Button
            variant="secondary"
            size="sm"
            loading={busy}
            onClick={() => void onDecide('DENIED')}
          >
            <Icon name="x" size={20} /> 거부
          </Button>
          <Button
            variant="primary"
            size="sm"
            loading={busy}
            onClick={() => void onDecide('APPROVED')}
          >
            <Icon name="check" size={20} /> 승인
          </Button>
        </div>
      ) : (
        <div className="approval-card-actions">
          <StatusChip
            tone={approval.status === 'APPROVED' ? 'verified' : 'pending'}
            label={STATUS_LABELS[approval.status]}
          />
        </div>
      )}
    </div>
  )
}

export default ApprovalCard
