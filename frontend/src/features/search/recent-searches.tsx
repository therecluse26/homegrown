import { useCallback, useSyncExternalStore } from "react";
import { FormattedMessage } from "react-intl";
import { Clock } from "lucide-react";
import { Button, Icon } from "@/components/ui";

// ─── Constants ───────────────────────────────────────────────────────────────

const STORAGE_KEY = "homegrown_recent_searches";
const MAX_RECENT = 10;

// ─── Storage utilities ───────────────────────────────────────────────────────

let listeners: Array<() => void> = [];

function emitChange() {
  for (const listener of listeners) {
    listener();
  }
}

function getSnapshot(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as string[]) : []; // safe: we only write string[] to this key
  } catch {
    return [];
  }
}

function getServerSnapshot(): string[] {
  return [];
}

function subscribe(listener: () => void): () => void {
  listeners = [...listeners, listener];
  return () => {
    listeners = listeners.filter((l) => l !== listener);
  };
}

/** Add a search query to the recent list. Exported for use by the parent search page. */
export function addRecentSearch(query: string): void {
  const trimmed = query.trim();
  if (!trimmed) return;

  const current = getSnapshot();
  // Remove duplicate if already present, then prepend
  const updated = [trimmed, ...current.filter((q) => q !== trimmed)].slice(
    0,
    MAX_RECENT,
  );

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
  } catch {
    // Storage full or unavailable — silently ignore
  }
  emitChange();
}

function clearRecentSearches(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Silently ignore
  }
  emitChange();
}

// ─── Types ───────────────────────────────────────────────────────────────────

type RecentSearchesProps = {
  onSelect: (query: string) => void;
};

// ─── Component ───────────────────────────────────────────────────────────────

export function RecentSearches({ onSelect }: RecentSearchesProps) {
  const searches = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  const handleClear = useCallback(() => {
    clearRecentSearches();
  }, []);

  if (searches.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="type-label-lg text-on-surface-variant">
          <FormattedMessage
            id="search.recentSearches.title"
            defaultMessage="Recent searches"
          />
        </h3>
        <Button variant="tertiary" size="sm" onClick={handleClear}>
          <FormattedMessage
            id="search.recentSearches.clear"
            defaultMessage="Clear history"
          />
        </Button>
      </div>

      <ul className="space-y-0.5">
        {searches.map((query) => (
          <li key={query}>
            <button
              type="button"
              onClick={() => onSelect(query)}
              className="flex w-full items-center gap-3 rounded-radius-md px-3 py-2 text-left transition-colors hover:bg-surface-container-low"
            >
              <Icon
                icon={Clock}
                size="sm"
                aria-hidden
                className="text-on-surface-variant shrink-0"
              />
              <span className="type-body-md text-on-surface truncate">
                {query}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
