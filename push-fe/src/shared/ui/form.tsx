import type { ReactNode } from 'react'

const Form = ({ children }: { children: ReactNode }) => (
  <div className="form">{children}</div>
)

const FormSection = ({
  title,
  children,
}: {
  title: string
  children: ReactNode
}) => (
  <div className="form-section">
    <div className="form-section-title">{title}</div>
    {children}
  </div>
)

const FormRow = ({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children?: ReactNode
}) => (
  <div className="form-row">
    <span className="form-row-label">{label}</span>
    {children ?? (hint ? <span className="form-row-hint">{hint}</span> : null)}
  </div>
)

export { Form, FormSection, FormRow }
