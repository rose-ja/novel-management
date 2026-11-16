"use client";

import { Check } from "lucide-react";

import { messages } from "@/lib/messages";

export type ImportStep = 1 | 2 | 3 | 4 | 5;

const steps: Array<{ id: ImportStep; label: string }> = [
  { id: 1, label: messages.importStepUpload },
  { id: 2, label: messages.importStepValidate },
  { id: 3, label: messages.importStepConfirm },
  { id: 4, label: messages.importStepReport },
  { id: 5, label: messages.importStepReview },
];

export function ImportStepper({ step, hasError }: { step: ImportStep; hasError?: boolean }) {
  return (
    <ol className="import-stepper" aria-label="导入步骤">
      {steps.map((item) => {
        const done = item.id < step;
        const current = item.id === step;
        const error = current && hasError;
        return (
          <li
            key={item.id}
            className={[
              "import-stepper__item",
              done ? "import-stepper__item--done" : "",
              current ? "import-stepper__item--current" : "",
              error ? "import-stepper__item--error" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            aria-current={current ? "step" : undefined}
          >
            <span className="import-stepper__num" aria-hidden="true">
              {done ? <Check size={14} /> : item.id}
            </span>
            <span className="import-stepper__label">{item.label}</span>
          </li>
        );
      })}
    </ol>
  );
}
