import { useRef, useState } from 'react'
import { COMPOSER_PLACEHOLDER } from '@/shared/constants'
import { IconButton, showToast } from '@/shared/ui'
import { uploadSource } from '@/shared/api'
import InferenceSettings from './inference-settings'

type ComposerProps = {
  onSend: (text: string) => void
  sending?: boolean
}

const Composer = ({ onSend, sending = false }: ComposerProps) => {
  const [text, setText] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const submit = () => {
    const value = text.trim()
    if (!value || sending) return
    onSend(value)
    setText('')
  }

  const onAttach = async (file: File | undefined) => {
    if (!file) return
    try {
      await uploadSource(file, 'RESUME')
      showToast('파일을 올렸어요')
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '업로드하지 못했어요',
        'circle-alert',
      )
    }
  }

  return (
    <div className="composer-wrap">
      {settingsOpen ? <InferenceSettings /> : null}
      <div className="composer">
        <IconButton
          icon="paperclip"
          aria-label="파일 첨부"
          onClick={() => fileRef.current?.click()}
        />
        <input
          ref={fileRef}
          type="file"
          hidden
          onChange={(e) => {
            void onAttach(e.target.files?.[0])
            e.target.value = ''
          }}
        />
        <IconButton
          icon="sliders-horizontal"
          aria-label="추론 설정"
          onClick={() => setSettingsOpen((v) => !v)}
        />
        <input
          className="composer-field"
          placeholder={COMPOSER_PLACEHOLDER}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.nativeEvent.isComposing) submit()
          }}
        />
        <IconButton
          icon="send-horizontal"
          iconSize={16}
          aria-label="보내기"
          onClick={submit}
          disabled={!text.trim() || sending}
        />
      </div>
    </div>
  )
}

export default Composer
