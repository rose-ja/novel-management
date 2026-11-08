export type SkeletonProps = {
  /** text：单行文本；block：卡片/图片；lines：多行文本。 */
  variant?: "text" | "block" | "lines";
  lines?: number;
  width?: string;
  height?: string;
  label?: string;
};

export function Skeleton({ variant = "text", lines = 3, width, height, label = "加载中" }: SkeletonProps) {
  const style = width || height ? { width, height } : undefined;

  if (variant === "lines") {
    return (
      <div className="ui-skeleton-lines" role="status" aria-label={label}>
        {Array.from({ length: lines }, (_, index) => (
          <span
            key={index}
            className="ui-skeleton ui-skeleton--text"
            style={{ width: index === lines - 1 ? "60%" : "100%" }}
          />
        ))}
        <span className="ui-visually-hidden">{label}</span>
      </div>
    );
  }

  return (
    <span
      className={variant === "block" ? "ui-skeleton ui-skeleton--block" : "ui-skeleton ui-skeleton--text"}
      style={style}
      role="status"
      aria-label={label}
    />
  );
}
