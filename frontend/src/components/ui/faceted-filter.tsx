import { type ReactNode } from "react";
import { Checkbox } from "./checkbox";

type FilterOption = {
  id: string;
  label: string;
  count?: number;
};

type FilterGroup = {
  id: string;
  label: string;
  options: FilterOption[];
};

type FacetedFilterProps = {
  groups: FilterGroup[];
  /** Currently selected option IDs, keyed by group ID */
  selected: Record<string, string[]>;
  /** Called when selection changes */
  onChange: (groupId: string, optionId: string, checked: boolean) => void;
  /** Optional header actions (e.g., "Clear all") */
  headerAction?: ReactNode;
  className?: string;
};

export function FacetedFilter({
  groups,
  selected,
  onChange,
  headerAction,
  className = "",
}: FacetedFilterProps) {
  return (
    <div className={`flex flex-col gap-6 ${className}`}>
      {headerAction && (
        <div className="flex items-center justify-between">
          <span className="type-title-sm text-on-surface">Filters</span>
          {headerAction}
        </div>
      )}

      {groups.map((group) => (
        <fieldset key={group.id} className="flex flex-col gap-2">
          <legend className="type-label-lg text-on-surface mb-1">
            {group.label}
          </legend>
          {group.options.map((option) => {
            const isSelected = selected[group.id]?.includes(option.id) ?? false;

            return (
              <div key={option.id} className="flex items-center justify-between">
                <Checkbox
                  label={option.label}
                  checked={isSelected}
                  onChange={(e) =>
                    onChange(group.id, option.id, e.currentTarget.checked)
                  }
                />
                {option.count !== undefined && (
                  <span className="type-label-sm text-on-surface-variant">
                    {option.count}
                  </span>
                )}
              </div>
            );
          })}
        </fieldset>
      ))}
    </div>
  );
}
