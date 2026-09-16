import type { ReactNode } from 'react'

const CardGrid = ({ children }: { children: ReactNode }) => (
  <div className="card-grid">{children}</div>
)

export default CardGrid
