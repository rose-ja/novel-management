import type { ReactNode } from "react";
import { AlertTriangle, CheckCircle2, Info, XCircle } from "lucide-react";

export type AlertTone = "info" | "success" | "warning" | "danger";

export type AlertProps = {
  tone?: AlertTone;
  title: string;
  children?: ReactNode;
  action?: ReactNode;
};

const toneIcon: Record<AlertTone, ReactNode> = {
  info: <Info aria-hidden="true" size={18} />,
  success: <CheckCircle2 aria-hidden="true" size={18} />,
  warning: <AlertTriangle aria-hidden="true" size={18} />,
  danger: <XCircle aria-hidden="true" size={18} />,
};

export function Alert({ tone = "info", title, children, action }: AlertProps) {
  const assertive = tone === "danger" || tone === "warning";

  return (
    <div className={`ui-alert ui-alert--${tone}`} role={assertive ? "alert" : "status"}>
      <span className="ui-alert__icon">{toneIcon[tone]}</span>
      <div className="ui-alert__content">
        <p className="ui-alert__title">{title}</p>
        {children ? <div className="ui-alert__body">{children}</div> : null}
      </div>
      {action ? <div className="ui-alert__action">{action}</div> : null}
    </div>
  );
}
