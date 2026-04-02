import { useCallback, useMemo } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { X } from "lucide-react";
import { Card, Input, Select, Button, Badge, Icon } from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import type {
  SearchScope,
  SearchParams,
  FacetCounts,
  FacetBucket,
} from "@/hooks/use-search";

// ─── Types ───────────────────────────────────────────────────────────────────

type SearchFiltersProps = {
  scope: SearchScope;
  facets?: FacetCounts;
  filters: Partial<SearchParams>;
  onChange: (filters: Partial<SearchParams>) => void;
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

function toggleArrayValue(arr: string[] | undefined, value: string): string[] {
  const current = arr ?? [];
  return current.includes(value)
    ? current.filter((v) => v !== value)
    : [...current, value];
}

// ─── Facet list ──────────────────────────────────────────────────────────────

function FacetGroup({
  label,
  buckets,
  selected,
  onToggle,
}: {
  label: string;
  buckets: FacetBucket[];
  selected: string[];
  onToggle: (value: string) => void;
}) {
  if (buckets.length === 0) return null;

  return (
    <fieldset className="space-y-2">
      <legend className="type-label-lg text-on-surface font-semibold mb-1">
        {label}
      </legend>
      {buckets.map((bucket) => (
        <label
          key={bucket.value}
          className="flex items-center justify-between gap-2 cursor-pointer select-none type-body-md text-on-surface hover:bg-surface-container-low rounded-radius-md px-2 py-1.5 transition-colors"
        >
          <span className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={selected.includes(bucket.value)}
              onChange={() => onToggle(bucket.value)}
              className="h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-radius-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22currentColor%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat hover:bg-surface-container-high checked:hover:bg-primary-container focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            />
            <span>{bucket.display_name}</span>
          </span>
          <Badge variant="default">{bucket.count}</Badge>
        </label>
      ))}
    </fieldset>
  );
}

// ─── Marketplace filters ─────────────────────────────────────────────────────

function MarketplaceFilters({
  facets,
  filters,
  onChange,
}: {
  facets?: FacetCounts;
  filters: Partial<SearchParams>;
  onChange: (filters: Partial<SearchParams>) => void;
}) {
  const intl = useIntl();

  return (
    <div className="space-y-6">
      {facets && (
        <>
          <FacetGroup
            label={intl.formatMessage({ id: "search.filters.methodologyTags", defaultMessage: "Methodology" })}
            buckets={facets.methodology_tags}
            selected={filters.methodology_tags ?? []}
            onToggle={(value) =>
              onChange({ methodology_tags: toggleArrayValue(filters.methodology_tags, value) })
            }
          />

          <FacetGroup
            label={intl.formatMessage({ id: "search.filters.subjectTags", defaultMessage: "Subject" })}
            buckets={facets.subject_tags}
            selected={filters.subject_tags ?? []}
            onToggle={(value) =>
              onChange({ subject_tags: toggleArrayValue(filters.subject_tags, value) })
            }
          />

          <FacetGroup
            label={intl.formatMessage({ id: "search.filters.contentType", defaultMessage: "Content Type" })}
            buckets={facets.content_type}
            selected={filters.content_type ? [filters.content_type] : []}
            onToggle={(value) =>
              onChange({
                content_type: filters.content_type === value ? undefined : value,
              })
            }
          />

          <FacetGroup
            label={intl.formatMessage({ id: "search.filters.worldviewTags", defaultMessage: "Worldview" })}
            buckets={facets.worldview_tags}
            selected={filters.worldview_tags ?? []}
            onToggle={(value) =>
              onChange({ worldview_tags: toggleArrayValue(filters.worldview_tags, value) })
            }
          />
        </>
      )}

      {/* Price range */}
      <fieldset className="space-y-2">
        <legend className="type-label-lg text-on-surface font-semibold mb-1">
          {intl.formatMessage({ id: "search.filters.priceRange", defaultMessage: "Price Range" })}
        </legend>
        <div className="flex items-center gap-2">
          <label className="flex flex-col gap-1">
            <span className="type-label-sm text-on-surface-variant">
              {intl.formatMessage({ id: "search.filters.priceMin", defaultMessage: "Min" })}
            </span>
            <Input
              type="number"
              min={0}
              step={1}
              value={filters.price_min ?? ""}
              onChange={(e) =>
                onChange({
                  price_min: e.target.value ? Number(e.target.value) : undefined,
                })
              }
              className="w-24"
            />
          </label>
          <span className="type-body-sm text-on-surface-variant mt-5">&ndash;</span>
          <label className="flex flex-col gap-1">
            <span className="type-label-sm text-on-surface-variant">
              {intl.formatMessage({ id: "search.filters.priceMax", defaultMessage: "Max" })}
            </span>
            <Input
              type="number"
              min={0}
              step={1}
              value={filters.price_max ?? ""}
              onChange={(e) =>
                onChange({
                  price_max: e.target.value ? Number(e.target.value) : undefined,
                })
              }
              className="w-24"
            />
          </label>
        </div>
      </fieldset>
    </div>
  );
}

// ─── Learning filters ────────────────────────────────────────────────────────

function LearningFilters({
  filters,
  onChange,
}: {
  filters: Partial<SearchParams>;
  onChange: (filters: Partial<SearchParams>) => void;
}) {
  const intl = useIntl();
  const { data: students } = useStudents();

  return (
    <div className="space-y-6">
      {/* Student selector */}
      <fieldset className="space-y-2">
        <legend className="type-label-lg text-on-surface font-semibold mb-1">
          {intl.formatMessage({ id: "search.filters.student", defaultMessage: "Student" })}
        </legend>
        <Select
          value={filters.student_id ?? ""}
          onChange={(e) =>
            onChange({ student_id: e.target.value || undefined })
          }
        >
          <option value="">
            {intl.formatMessage({ id: "search.filters.allStudents", defaultMessage: "All students" })}
          </option>
          {students?.map((student) => (
            <option key={student.id} value={student.id}>
              {student.display_name}
            </option>
          ))}
        </Select>
      </fieldset>

      {/* Source type */}
      <fieldset className="space-y-2">
        <legend className="type-label-lg text-on-surface font-semibold mb-1">
          {intl.formatMessage({ id: "search.filters.sourceType", defaultMessage: "Source" })}
        </legend>
        <Select
          value={filters.source_type ?? ""}
          onChange={(e) =>
            onChange({ source_type: e.target.value || undefined })
          }
        >
          <option value="">
            {intl.formatMessage({ id: "search.filters.allSources", defaultMessage: "All sources" })}
          </option>
          {["activity", "journal", "reading"].map((t) => (
            <option key={t} value={t}>
              {intl.formatMessage({
                id: `search.filters.source.${t}`,
                defaultMessage: t.charAt(0).toUpperCase() + t.slice(1),
              })}
            </option>
          ))}
        </Select>
      </fieldset>

      {/* Date range */}
      <fieldset className="space-y-2">
        <legend className="type-label-lg text-on-surface font-semibold mb-1">
          {intl.formatMessage({ id: "search.filters.dateRange", defaultMessage: "Date Range" })}
        </legend>
        <div className="flex flex-col gap-2">
          <label className="flex flex-col gap-1">
            <span className="type-label-sm text-on-surface-variant">
              {intl.formatMessage({ id: "search.filters.dateFrom", defaultMessage: "From" })}
            </span>
            <Input
              type="date"
              value={filters.date_from ?? ""}
              onChange={(e) =>
                onChange({ date_from: e.target.value || undefined })
              }
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="type-label-sm text-on-surface-variant">
              {intl.formatMessage({ id: "search.filters.dateTo", defaultMessage: "To" })}
            </span>
            <Input
              type="date"
              value={filters.date_to ?? ""}
              onChange={(e) =>
                onChange({ date_to: e.target.value || undefined })
              }
            />
          </label>
        </div>
      </fieldset>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function SearchFilters({ scope, facets, filters, onChange }: SearchFiltersProps) {
  const hasActiveFilters = useMemo(() => {
    const {
      methodology_tags,
      subject_tags,
      content_type,
      worldview_tags,
      price_min,
      price_max,
      student_id,
      source_type,
      date_from,
      date_to,
    } = filters;

    return !!(
      (methodology_tags && methodology_tags.length > 0) ||
      (subject_tags && subject_tags.length > 0) ||
      content_type ||
      (worldview_tags && worldview_tags.length > 0) ||
      price_min != null ||
      price_max != null ||
      student_id ||
      source_type ||
      date_from ||
      date_to
    );
  }, [filters]);

  const handleClearAll = useCallback(() => {
    onChange({
      methodology_tags: undefined,
      subject_tags: undefined,
      content_type: undefined,
      worldview_tags: undefined,
      price_min: undefined,
      price_max: undefined,
      student_id: undefined,
      source_type: undefined,
      date_from: undefined,
      date_to: undefined,
    });
  }, [onChange]);

  return (
    <Card className="p-card-padding">
      <div className="flex items-center justify-between mb-4">
        <h2 className="type-title-sm text-on-surface">
          <FormattedMessage id="search.filters.title" defaultMessage="Filters" />
        </h2>
        {hasActiveFilters && (
          <Button
            variant="tertiary"
            size="sm"
            leadingIcon={<Icon icon={X} size="xs" aria-hidden />}
            onClick={handleClearAll}
          >
            <FormattedMessage id="search.filters.clearAll" defaultMessage="Clear all" />
          </Button>
        )}
      </div>

      {scope === "marketplace" && (
        <MarketplaceFilters facets={facets} filters={filters} onChange={onChange} />
      )}

      {scope === "learning" && (
        <LearningFilters filters={filters} onChange={onChange} />
      )}
    </Card>
  );
}
