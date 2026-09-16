import type { ReactNode } from 'react'

type DialogProps = {
  open: boolean
  title: string
  children: ReactNode
  actions?: ReactNode
  onClose?: () => void
}

const Dialog = ({ open, title, children, actions, onClose }: DialogProps) => {
  if (!open) return null
  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="dialog-title">{title}</div>
        {children}
        {actions ? <div className="dialog-actions">{actions}</div> : null}
      </div>
    </div>
  )
}

export default Dialog
