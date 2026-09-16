import { useState } from 'react'
import type { FormEvent } from 'react'
import type { JobPosting } from '@/shared/api'
import { createJob } from '@/shared/api'
import { Button, Dialog, Input, Select, Textarea, showToast } from '@/shared/ui'
import { isHttpUrl } from '@/shared/lib/url'

type JobAddDialogProps = {
  open: boolean
  onClose: () => void
  onCreated?: (job: JobPosting) => void
  initialSource?: string
}

const JobAddDialog = ({
  open,
  onClose,
  onCreated,
  initialSource = '',
}: JobAddDialogProps) => {
  const [company, setCompany] = useState('')
  const [title, setTitle] = useState('')
  const [source, setSource] = useState(initialSource)
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!company.trim() || !title.trim() || !source.trim()) return
    setBusy(true)
    try {
      const url = isHttpUrl(source)
      const job = await createJob({
        company: company.trim(),
        title: title.trim(),
        sourceKind: url ? 'URL' : 'TEXT',
        ...(url ? { sourceUrl: source.trim() } : {}),
        sourceText: url ? '' : source.trim(),
        requirements: [],
        preferred: [],
      })
      showToast('공고를 추가했어요')
      onCreated?.(job)
      onClose()
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '추가하지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      title="공고 추가"
      onClose={onClose}
      actions={
        <>
          <Button variant="secondary" size="sm" onClick={onClose}>
            취소
          </Button>
          <Button
            variant="primary"
            size="sm"
            loading={busy}
            onClick={(e) => void submit(e)}
          >
            추가
          </Button>
        </>
      }
    >
      <form
        className="form"
        onSubmit={(e) => void submit(e)}
        style={{ gap: 'var(--space-12)' }}
      >
        <Input
          placeholder="회사명"
          value={company}
          onChange={(e) => setCompany(e.target.value)}
          required
        />
        <Input
          placeholder="직무"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
        />
        <Select
          value={isHttpUrl(source) ? 'URL' : 'TEXT'}
          onChange={() => undefined}
          aria-label="수집 방식"
          style={{ display: 'none' }}
        />
        <Textarea
          placeholder="공고 URL 또는 공고 원문을 붙여넣으세요"
          value={source}
          onChange={(e) => setSource(e.target.value)}
          rows={4}
          required
        />
      </form>
    </Dialog>
  )
}

export default JobAddDialog
