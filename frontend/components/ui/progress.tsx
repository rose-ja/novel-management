export type ProgressProps = {
  /** 0-100；未提供时渲染不确定进度。 */
  value?: number;
  label?: string;
  tone?: "brand" | "success" | "warning" | "danger";
};

export function Progress({ value, label, tone = "brand" }: ProgressProps) {
  const indeterminate = typeof value !== "number";
  const safeValue = indeterminate ? 0 : Math.min(100, Math.max(0, value));

  return (
    <div className="ui-progress">
      {label ? (
        <div className="ui-progress__meta">
          <span>{label}</span>
          {indeterminate ? null : <span>{Math.round(safeValue)}%</span>}
        </div>
      ) : null}
      <div
        className={`ui-progress__track ui-progress__track--${tone}${indeterminate ? " ui-progress__track--indeterminate" : ""}`}
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={indeterminate ? undefined : safeValue}
        aria-label={label ?? "进度"}
      >
        <span className="ui-progress__bar" style={indeterminate ? undefined : { width: `${safeValue}%` }} />
      </div>
    </div>
  );
}
