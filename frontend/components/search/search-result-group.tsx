import { Badge } from "@/components/ui/badge";
import type { SearchGroup } from "@/lib/api/search";
import { entityTypeLabel } from "@/lib/search";

import { SearchResultRow } from "./search-result-row";

export function SearchResultGroup({ group }: { group: SearchGroup }) {
  const headingId = `search-group-${group.entity_type}`;
  return (
    <section className="search-group" aria-labelledby={headingId}>
      <header className="search-group__header">
        <h2 id={headingId} className="search-group__title">
          {entityTypeLabel(group.entity_type)}
        </h2>
        <Badge tone="brand">{group.total}</Badge>
      </header>
      <div className="search-group__items">
        {group.items.map((item) => (
          <SearchResultRow key={item.entity_id} item={item} />
        ))}
      </div>
    </section>
  );
}
