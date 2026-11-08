import Link from "next/link";
import { Compass } from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { messages } from "@/lib/messages";

export default function NotFound() {
  return (
    <div className="page">
      <EmptyState
        variant="page"
        icon={<Compass aria-hidden="true" size={22} />}
        title={messages.notFoundTitle}
        description={messages.notFoundDescription}
        action={
          <Link className="ui-button ui-button--primary ui-button--md" href="/">
            {messages.backHome}
          </Link>
        }
      />
    </div>
  );
}
