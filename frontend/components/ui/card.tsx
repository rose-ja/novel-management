import type { ReactNode } from "react";

export type CardProps = {
  title?: string;
  description?: string;
  extra?: ReactNode;
  footer?: ReactNode;
  children?: ReactNode;
  /** flat 用于页面分区，raised 用于需要层级的内容块。 */
  variant?: "flat" | "raised";
};

export function Card({ title, description, extra, footer, children, variant = "flat" }: CardProps) {
  return (
    <section className={`ui-card ui-card--${variant}`}>
      {title || extra ? (
        <header className="ui-card__header">
          <div>
            {title ? <h2 className="ui-card__title">{title}</h2> : null}
            {description ? <p className="ui-card__description">{description}</p> : null}
          </div>
          {extra ? <div className="ui-card__extra">{extra}</div> : null}
        </header>
      ) : null}
      {children ? <div className="ui-card__body">{children}</div> : null}
      {footer ? <footer className="ui-card__footer">{footer}</footer> : null}
    </section>
  );
}
