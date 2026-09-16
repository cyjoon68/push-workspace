import type { ReactNode } from 'react'

type CardProps = {
  title?: string
  meta?: string
  children?: ReactNode
  onClick?: () => void
}

const Card = ({ title, meta, children, onClick }: CardProps) => {
  if (onClick) {
    return (
      <button type="button" className="card card-clickable" onClick={onClick}>
        {title ? <div className="card-title">{title}</div> : null}
        {meta ? <div className="card-meta">{meta}</div> : null}
        {children}
      </button>
    )
  }
  return (
    <div className="card">
      {title ? <div className="card-title">{title}</div> : null}
      {meta ? <div className="card-meta">{meta}</div> : null}
      {children}
    </div>
  )
}

export default Card
