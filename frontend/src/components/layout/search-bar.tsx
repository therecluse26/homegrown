import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { useNavigate } from "react-router";
import { useIntl } from "react-intl";
import { Search, X } from "lucide-react";
import { Icon } from "@/components/ui";
import { useAutocomplete } from "@/hooks/use-search";
import type { AutocompleteSuggestion } from "@/hooks/use-search";

// ─── Debounce hook ──────────────────────────────────────────────────────────

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}

// ─── Search bar ─────────────────────────────────────────────────────────────

export function SearchBar() {
  const intl = useIntl();
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [query, setQuery] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const debouncedQuery = useDebouncedValue(query, 300);
  const { data: autocomplete } = useAutocomplete(debouncedQuery);
  const suggestions = useMemo(
    () => autocomplete?.suggestions?.slice(0, 5) ?? [],
    [autocomplete],
  );

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Global Ctrl+K to focus search
  useEffect(() => {
    function handleGlobalKey(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        inputRef.current?.focus();
        setIsOpen(true);
      }
    }
    document.addEventListener("keydown", handleGlobalKey);
    return () => document.removeEventListener("keydown", handleGlobalKey);
  }, []);

  const navigateToSearch = useCallback(
    (searchQuery: string) => {
      if (!searchQuery.trim()) return;
      setIsOpen(false);
      setQuery("");
      navigate(`/search?q=${encodeURIComponent(searchQuery.trim())}`);
    },
    [navigate],
  );

  const handleSelect = useCallback(
    (suggestion: AutocompleteSuggestion) => {
      setIsOpen(false);
      setQuery("");
      navigate(`/search?q=${encodeURIComponent(suggestion.text ?? "")}`);
    },
    [navigate],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev < suggestions.length - 1 ? prev + 1 : prev,
          );
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) => (prev > 0 ? prev - 1 : -1));
          break;
        case "Enter":
          e.preventDefault();
          if (selectedIndex >= 0 && suggestions[selectedIndex]) {
            handleSelect(suggestions[selectedIndex]);
          } else {
            navigateToSearch(query);
          }
          break;
        case "Escape":
          setIsOpen(false);
          inputRef.current?.blur();
          break;
      }
    },
    [suggestions, selectedIndex, query, handleSelect, navigateToSearch],
  );

  const showSuggestions = isOpen && suggestions.length > 0 && query.length >= 1;

  return (
    <div ref={containerRef} className="relative hidden md:block">
      <div className="relative">
        <Icon
          icon={Search}
          size="sm"
          className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none"
        />
        <input
          ref={inputRef}
          type="search"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setIsOpen(true);
            setSelectedIndex(-1);
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={intl.formatMessage({ id: "search.bar.placeholder" })}
          className="w-56 lg:w-72 pl-9 pr-8 py-2 bg-surface-container-highest rounded-radius-md text-on-surface type-body-sm placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset transition-all duration-[var(--duration-normal)]"
          role="combobox"
          aria-expanded={showSuggestions}
          aria-controls="search-suggestions"
          aria-activedescendant={
            selectedIndex >= 0 ? `search-suggestion-${selectedIndex}` : undefined
          }
          aria-label={intl.formatMessage({ id: "search.bar.label" })}
        />
        {query && (
          <button
            onClick={() => {
              setQuery("");
              setIsOpen(false);
              inputRef.current?.focus();
            }}
            className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded-radius-sm text-on-surface-variant hover:text-on-surface transition-colors"
            aria-label={intl.formatMessage({ id: "search.bar.clear" })}
          >
            <Icon icon={X} size="xs" />
          </button>
        )}
      </div>

      {/* Suggestions dropdown */}
      {showSuggestions && (
        <ul
          id="search-suggestions"
          role="listbox"
          className="absolute top-full left-0 right-0 mt-1 bg-surface-container-lowest rounded-radius-md shadow-ambient-md z-[var(--z-popover)] overflow-hidden"
        >
          {suggestions.map((suggestion, index) => (
            <li
              key={suggestion.entity_id}
              id={`search-suggestion-${index}`}
              role="option"
              aria-selected={index === selectedIndex}
              onClick={() => handleSelect(suggestion)}
              onMouseEnter={() => setSelectedIndex(index)}
              className={`flex items-center gap-2 px-3 py-2.5 cursor-pointer type-body-sm transition-colors ${
                index === selectedIndex
                  ? "bg-surface-container-high text-on-surface"
                  : "text-on-surface-variant hover:bg-surface-container-low"
              }`}
            >
              <Icon icon={Search} size="xs" className="shrink-0 opacity-50" />
              <span className="truncate">{suggestion.text}</span>
              <span className="ml-auto type-label-sm text-on-surface-variant/60 shrink-0">
                {suggestion.entity_type}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
