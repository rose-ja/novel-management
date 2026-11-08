import type { ReactNode } from "react";

export type EmptyStateProps = {
  title: string;
  description: string;
  /** page：整页空；section：卡片内；inline：表格/列表内。 */
  variant?: "page" | "section" | "inline";
  icon?: ReactNode;
  action?: ReactNode;
};

/** 空状态必须给出原因与下一步；没有可用操作时不渲染按钮。 */
export function EmptyState({ title, description, variant = "section", icon, action }: EmptyStateProps) {
  return (
    <div className={`ui-empty ui-empty--${variant}`}>
      {icon ? <div className="ui-empty__icon">{icon}</div> : null}
      <h2 className="ui-empty__title">{title}</h2>
      <p className="ui-empty__description">{description}</p>
      {action ? <div className="ui-empty__action">{action}</div> : null}
    </div>
  );
}
