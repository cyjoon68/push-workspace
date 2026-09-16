import { useState } from 'react'
import { COMMAND_INPUT_PLACEHOLDER } from '@/shared/constants'
import { Button, Icon, IconButton } from '@/shared/ui'

type CommandInputProps = {
  onSubmit: (text: string) => void
  busy?: boolean
}

const CommandInput = ({ onSubmit, busy = false }: CommandInputProps) => {
  const [text, setText] = useState('')

  const submit = () => {
    const value = text.trim()
    if (!value || busy) return
    onSubmit(value)
    setText('')
  }

  return (
    <div className="command-input home-command">
      <IconButton icon="paperclip" aria-label="파일 첨부" />
      <input
        className="command-input-field"
        placeholder={COMMAND_INPUT_PLACEHOLDER}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.nativeEvent.isComposing) submit()
        }}
      />
      <Button
        variant="primary"
        size="md"
        loading={busy}
        onClick={submit}
        icon={<Icon name="corner-down-left" size={16} />}
      >
        실행
      </Button>
    </div>
  )
}

export default CommandInput
