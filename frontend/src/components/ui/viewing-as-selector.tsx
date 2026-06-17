import { Eye } from "lucide-react";
import { useIntl } from "react-intl";
import { Icon } from "@/components/ui/icon";
import { useStudents } from "@/hooks/use-family";

type ViewingAsSelectorProps = {
  /** Currently selected student ID, or undefined for "no child selected". */
  value: string | undefined;
  onChange: (studentId: string | undefined) => void;
  className?: string;
};

/**
 * Inline child-selector dropdown for fit-badge–enabled browse surfaces.
 * Shows "Viewing as: [child]" — selecting a child triggers the parent to pass
 * `for_student_id` to the listing/activity-def/reading-item API.
 *
 * Renders nothing when the family has no students.
 */
export function ViewingAsSelector({
  value,
  onChange,
  className = "",
}: ViewingAsSelectorProps) {
  const intl = useIntl();
  const { data: students } = useStudents();

  if (!students || students.length === 0) return null;

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <Icon icon={Eye} size="sm" className="text-on-surface-variant shrink-0" aria-hidden />
      <label
        htmlFor="viewing-as-selector"
        className="type-label-sm text-on-surface-variant shrink-0"
      >
        {intl.formatMessage({
          id: "browse.viewingAs.label",
          defaultMessage: "Viewing as:",
        })}
      </label>
      <select
        id="viewing-as-selector"
        value={value ?? ""}
        onChange={(e) => onChange(e.target.value || undefined)}
        className="bg-surface-container-highest rounded-sm px-3 py-1.5 text-on-surface type-body-sm border-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <option value="">
          {intl.formatMessage({
            id: "browse.viewingAs.none",
            defaultMessage: "Everyone",
          })}
        </option>
        {students.map((s) => (
          <option key={s.id} value={s.id}>
            {s.display_name}
          </option>
        ))}
      </select>
    </div>
  );
}
