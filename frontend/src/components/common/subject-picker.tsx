import { useState, useCallback, useId } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { ChevronRight, Plus, Tag } from "lucide-react";
import { Button, Icon, Input, Spinner } from "@/components/ui";
import {
  useSubjectTaxonomy,
  useCreateCustomSubject,
  type SubjectTaxonomyResponse,
} from "@/hooks/use-subjects";

// ─── Props ──────────────────────────────────────────────────────────────────

interface SubjectPickerProps {
  /** Currently selected subject slugs */
  value: string[];
  /** Called when selection changes */
  onChange: (slugs: string[]) => void;
  /** Allow creating custom subjects inline */
  allowCustom?: boolean;
  /** Max selections allowed (0 = unlimited) */
  max?: number;
}

// ─── Recursive tree node ────────────────────────────────────────────────────

function TaxonomyNode({
  node,
  selected,
  onToggle,
  depth,
}: {
  node: SubjectTaxonomyResponse;
  selected: Set<string>;
  onToggle: (slug: string) => void;
  depth: number;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasChildren = node.children.length > 0;
  const isSelected = selected.has(node.slug);
  const checkboxId = useId();

  return (
    <li role="treeitem" aria-expanded={hasChildren ? expanded : undefined}>
      <div
        className="flex items-center gap-2 py-1.5 rounded-lg hover:bg-surface-container-low transition-colors"
        style={{ paddingLeft: `${depth * 1.25}rem` }}
      >
        {hasChildren ? (
          <button
            type="button"
            onClick={() => setExpanded((prev) => !prev)}
            className="shrink-0 p-0.5 rounded text-on-surface-variant hover:text-on-surface transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring"
            aria-label={expanded ? "Collapse" : "Expand"}
          >
            <Icon
              icon={ChevronRight}
              size="sm"
              className={`transition-transform ${expanded ? "rotate-90" : ""}`}
            />
          </button>
        ) : (
          <span className="w-5 shrink-0" />
        )}
        <input
          id={checkboxId}
          type="checkbox"
          checked={isSelected}
          onChange={() => onToggle(node.slug)}
          className="shrink-0 w-4 h-4 accent-primary rounded cursor-pointer"
        />
        <label
          htmlFor={checkboxId}
          className={`type-body-sm cursor-pointer flex items-center gap-1.5 ${
            isSelected ? "text-on-surface font-medium" : "text-on-surface-variant"
          }`}
        >
          {node.name}
          {node.is_custom && (
            <span className="type-label-sm text-secondary px-1.5 py-0.5 bg-secondary-container rounded-full">
              <FormattedMessage id="subjectPicker.custom" />
            </span>
          )}
        </label>
      </div>
      {hasChildren && expanded && (
        <ul role="group" className="list-none">
          {node.children.map((child) => (
            <TaxonomyNode
              key={child.id}
              node={child}
              selected={selected}
              onToggle={onToggle}
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

// ─── Main component ─────────────────────────────────────────────────────────

export function SubjectPicker({
  value,
  onChange,
  allowCustom = true,
  max = 0,
}: SubjectPickerProps) {
  const intl = useIntl();
  const { data: taxonomy, isPending } = useSubjectTaxonomy();
  const createCustom = useCreateCustomSubject();
  const [newSubjectName, setNewSubjectName] = useState("");
  const [showCustomForm, setShowCustomForm] = useState(false);

  const selectedSet = new Set(value);

  const handleToggle = useCallback(
    (slug: string) => {
      if (selectedSet.has(slug)) {
        onChange(value.filter((s) => s !== slug));
      } else if (max === 0 || value.length < max) {
        onChange([...value, slug]);
      }
    },
    [value, onChange, max, selectedSet],
  );

  function handleCreateCustom() {
    const name = newSubjectName.trim();
    if (!name) return;
    createCustom.mutate(
      { name },
      {
        onSuccess: (data) => {
          setNewSubjectName("");
          setShowCustomForm(false);
          onChange([...value, data.slug]);
        },
      },
    );
  }

  if (isPending) {
    return (
      <div className="flex items-center gap-2 py-4">
        <Spinner size="sm" />
        <span className="type-body-sm text-on-surface-variant">
          <FormattedMessage id="subjectPicker.loading" />
        </span>
      </div>
    );
  }

  return (
    <div>
      {/* Selected tags summary */}
      {value.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-3">
          {value.map((slug) => (
            <span
              key={slug}
              className="inline-flex items-center gap-1 px-2 py-1 bg-primary-container text-on-primary-container type-label-sm rounded-full"
            >
              <Icon icon={Tag} size="xs" aria-hidden />
              {slug}
              <button
                type="button"
                onClick={() => handleToggle(slug)}
                className="ml-0.5 hover:text-error transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring rounded-full"
                aria-label={intl.formatMessage(
                  { id: "subjectPicker.remove" },
                  { subject: slug },
                )}
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Taxonomy tree */}
      <div className="max-h-64 overflow-y-auto rounded-lg bg-surface-container-lowest p-2">
        {taxonomy && taxonomy.length > 0 ? (
          <ul role="tree" className="list-none">
            {taxonomy.map((node) => (
              <TaxonomyNode
                key={node.id}
                node={node}
                selected={selectedSet}
                onToggle={handleToggle}
                depth={0}
              />
            ))}
          </ul>
        ) : (
          <p className="type-body-sm text-on-surface-variant text-center py-4">
            <FormattedMessage id="subjectPicker.empty" />
          </p>
        )}
      </div>

      {/* Custom subject creation */}
      {allowCustom && (
        <div className="mt-2">
          {showCustomForm ? (
            <div className="flex items-center gap-2">
              <Input
                value={newSubjectName}
                onChange={(e) => setNewSubjectName(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "subjectPicker.customPlaceholder",
                })}
                className="flex-1"
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleCreateCustom();
                  }
                }}
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={handleCreateCustom}
                loading={createCustom.isPending}
                disabled={!newSubjectName.trim()}
              >
                <FormattedMessage id="subjectPicker.add" />
              </Button>
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => {
                  setShowCustomForm(false);
                  setNewSubjectName("");
                }}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
            </div>
          ) : (
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => setShowCustomForm(true)}
            >
              <Icon icon={Plus} size="sm" aria-hidden />
              <span className="ml-1">
                <FormattedMessage id="subjectPicker.createCustom" />
              </span>
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
