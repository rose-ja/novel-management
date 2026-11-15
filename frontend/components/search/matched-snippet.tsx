import type { SearchMatchedField } from "@/lib/api/search";
import { matchedFieldLabel } from "@/lib/search";

export function MatchedSnippet({ fields }: { fields: SearchMatchedField[] }) {
  if (fields.length === 0) return null;
  return (
    <div className="search-result__matches">
      {fields.map((field) => (
        <span key={field.field} className="search-result__match">
          <span className="search-result__match-label">{matchedFieldLabel(field.field)}：</span>
          <mark className="search-mark">{field.matched_text}</mark>
        </span>
      ))}
    </div>
  );
}
