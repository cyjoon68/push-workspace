const Skeleton = ({ height = 16 }: { height?: number }) => (
  <div className="skeleton" style={{ height }} />
)

const SkeletonCardGrid = ({ count = 3 }: { count?: number }) => (
  <div className="card-grid">
    {Array.from({ length: count }, (_, i) => (
      <Skeleton key={i} height={96} />
    ))}
  </div>
)

const SkeletonRows = ({ count = 4, height = 56 }: { count?: number; height?: number }) => (
  <>
    {Array.from({ length: count }, (_, i) => (
      <Skeleton key={i} height={height} />
    ))}
  </>
)

export { Skeleton, SkeletonCardGrid, SkeletonRows }
