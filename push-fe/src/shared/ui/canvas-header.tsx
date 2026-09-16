import type { ReactNode } from 'react'

type CanvasHeaderProps = {
  title: string
  actions?: ReactNode
  headingLevel?: 1 | 2
}

const CanvasHeader = ({
  title,
  actions,
  headingLevel = 1,
}: CanvasHeaderProps) => (
  <div className="canvas-header">
    {headingLevel === 1 ? (
      <h1 className="canvas-header-title">{title}</h1>
    ) : (
      <h2 className="canvas-header-title">{title}</h2>
    )}
    <div className="canvas-header-actions">{actions}</div>
  </div>
)

export default CanvasHeader
