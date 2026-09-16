import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { createDocumentExport } from '@/shared/api'
import {
  Button,
  CanvasHeader,
  ErrorState,
  Icon,
  SkeletonRows,
  showToast,
} from '@/shared/ui'
import { useDocumentsStore } from '@/entities/document'
import { RENDERER_VERSION } from '../constants'

const DocEditorPage = () => {
  const { id = '' } = useParams()
  const current = useDocumentsStore((s) => s.current)
  const currentVersion = useDocumentsStore((s) => s.currentVersion)
  const status = useDocumentsStore((s) => s.currentStatus)
  const error = useDocumentsStore((s) => s.currentError)
  const loadOne = useDocumentsStore((s) => s.loadOne)
  const resetCurrent = useDocumentsStore((s) => s.resetCurrent)
  const [exporting, setExporting] = useState<'PDF' | 'DOCX' | null>(null)

  const editor = useEditor({
    extensions: [StarterKit],
    editorProps: { attributes: { class: 'editor-body' } },
  })

  useEffect(() => {
    void loadOne(id)
    return () => resetCurrent()
  }, [id, loadOne, resetCurrent])

  useEffect(() => {
    if (editor && currentVersion) {
      editor.commands.setContent(currentVersion.content)
    }
  }, [editor, currentVersion])

  const exportAs = async (format: 'PDF' | 'DOCX') => {
    if (!current || !currentVersion) return
    setExporting(format)
    try {
      await createDocumentExport(current.id, {
        versionId: currentVersion.id,
        format,
        rendererVersion: RENDERER_VERSION,
      })
      showToast(`${format} 출력을 준비했어요`, 'file-down')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '출력하지 못했어요',
        'circle-alert',
      )
    } finally {
      setExporting(null)
    }
  }

  return (
    <>
      <CanvasHeader
        title={current?.title ?? '문서'}
        actions={
          <>
            <Button
              variant="secondary"
              size="sm"
              disabled={!currentVersion}
              loading={exporting === 'DOCX'}
              onClick={() => void exportAs('DOCX')}
            >
              <Icon name="file-down" size={20} /> DOCX
            </Button>
            <Button
              variant="primary"
              size="sm"
              disabled={!currentVersion}
              loading={exporting === 'PDF'}
              onClick={() => void exportAs('PDF')}
            >
              <Icon name="file-down" size={20} /> PDF
            </Button>
          </>
        }
      />
      {status === 'error' ? (
        <div className="canvas-body">
          <ErrorState
            message={error ?? undefined}
            onRetry={() => void loadOne(id)}
          />
        </div>
      ) : (
        <div className="editor">
          {status === 'loading' || status === 'idle' ? (
            <div className="editor-body">
              <SkeletonRows count={6} height={20} />
            </div>
          ) : (
            <EditorContent editor={editor} />
          )}
        </div>
      )}
    </>
  )
}

export default DocEditorPage
