import { useCallback, useEffect, useRef, useState } from "react";
import { useIntl } from "react-intl";
import { Search } from "lucide-react";
import { Badge, Icon, Spinner } from "@/components/ui";
import { useAutocomplete } from "@/hooks/use-search";
import type { SearchScope } from "@/hooks/use-search";

// ─── Types ───────────────────────────────────────────────────────────────────

type SearchSuggestionsProps = {
  query: string;
  scope?: SearchScope;
  onSelect: (text: string) => void;
  visible: boolean;
};

// ─── Entity type → variant mapping ───────────────────────────────────────────

const ENTITY_VARIANT: Record<string, "primary" | "secondary" | "default"> = {
  family: "primary",
  group: "primary",
  listing: "secondary",
  activity: "default",
  journal: "default",
  event: "secondary",
};

// ─── Component ───────────────────────────────────────────────────────────────

export function SearchSuggestions({
  query,
  scope,
  onSelect,
  visible,
}: SearchSuggestionsProps) {
  const intl = useIntl();
  const listRef = useRef<HTMLUListElement>(null);
  const [activeIndex, setActiveIndex] = useState(-1);

  const { data, isPending } = useAutocomplete(
    visible ? query : "",
    scope,
  );

  const suggestions = data?.suggestions ?? [];

  // Reset active index when suggestions change
  useEffect(() => {
    setActiveIndex(-1);
  }, [suggestions.length, query]);

  // Keyboard navigation handler — meant to be called from the parent input's onKeyDown
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!visible || suggestions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setActiveIndex((prev) =>
            prev < suggestions.length - 1 ? prev + 1 : 0,
          );
          break;
        case "ArrowUp":
          e.preventDefault();
          setActiveIndex((prev) =>
            prev > 0 ? prev - 1 : suggestions.length - 1,
          );
          break;
        case "Enter": {
          const selected = suggestions[activeIndex];
          if (activeIndex >= 0 && selected) {
            e.preventDefault();
            onSelect(selected.text);
          }
        }
          break;
        case "Escape":
          setActiveIndex(-1);
          break;
      }
    },
    [visible, suggestions, activeIndex, onSelect],
  );

  // Scroll active item into view
  useEffect(() => {
    if (activeIndex < 0 || !listRef.current) return;
    const items = listRef.current.querySelectorAll("[role='option']");
    items[activeIndex]?.scrollIntoView({ block: "nearest" });
  }, [activeIndex]);

  if (!visible || query.length < 1) return null;

  return (
    <div
      className="absolute left-0 right-0 top-full z-popover mt-1 overflow-hidden rounded-radius-md bg-surface-container-lowest shadow-ambient-md"
      role="listbox"
      aria-label={intl.formatMessage({
        id: "search.suggestions.label",
        defaultMessage: "Search suggestions",
      })}
      onKeyDown={handleKeyDown}
    >
      {isPending && (
        <div className="flex items-center justify-center py-4">
          <Spinner size="sm" />
        </div>
      )}

      {!isPending && suggestions.length === 0 && query.length >= 1 && (
        <div className="px-4 py-3 type-body-sm text-on-surface-variant">
          {intl.formatMessage({
            id: "search.suggestions.empty",
            defaultMessage: "No suggestions found",
          })}
        </div>
      )}

      {!isPending && suggestions.length > 0 && (
        <ul ref={listRef} className="max-h-72 overflow-y-auto py-1">
          {suggestions.map((suggestion, index) => (
            <li
              key={`${suggestion.entity_id}-${index}`}
              role="option"
              aria-selected={index === activeIndex}
              className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors ${
                index === activeIndex
                  ? "bg-surface-container-high"
                  : "hover:bg-surface-container-low"
              }`}
              onClick={() => onSelect(suggestion.text)}
              onMouseEnter={() => setActiveIndex(index)}
            >
              <Icon
                icon={Search}
                size="sm"
                className="text-on-surface-variant shrink-0"
              />
              <span className="type-body-md text-on-surface flex-1 min-w-0 truncate">
                {suggestion.text}
              </span>
              <Badge
                variant={ENTITY_VARIANT[suggestion.entity_type] ?? "default"}
              >
                {intl.formatMessage({
                  id: `search.entityType.${suggestion.entity_type}`,
                  defaultMessage: suggestion.entity_type,
                })}
              </Badge>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
