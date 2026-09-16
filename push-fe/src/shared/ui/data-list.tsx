import type { ReactNode } from 'react'

const DataList = ({ children }: { children: ReactNode }) => (
  <div className="data-list">{children}</div>
)

type DataListRowProps = {
  title: string
  meta?: string
  trailing?: ReactNode
  onClick?: () => void
}

const DataListRow = ({ title, meta, trailing, onClick }: DataListRowProps) => {
  const inner = (
    <>
      <div className="data-list-main">
        <div className="t-body">{title}</div>
        {meta ? <div className="card-meta">{meta}</div> : null}
      </div>
      {trailing}
    </>
  )
  if (onClick) {
    return (
      <button
        type="button"
        className="data-list-row data-list-row-button"
        onClick={onClick}
      >
        {inner}
      </button>
    )
  }
  return <div className="data-list-row">{inner}</div>
}

export { DataList, DataListRow }
