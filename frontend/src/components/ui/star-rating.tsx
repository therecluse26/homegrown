import { useState, useCallback, type KeyboardEvent } from "react";
import { Star } from "lucide-react";

type StarRatingProps = {
  /** Current value (1–5) */
  value: number;
  /** Called when the user selects a rating */
  onChange?: (value: number) => void;
  /** Read-only display mode */
  readOnly?: boolean;
  /** Accessible label */
  label?: string;
  className?: string;
};

export function StarRating({
  value,
  onChange,
  readOnly = false,
  label = "Rating",
  className = "",
}: StarRatingProps) {
  const [hoverValue, setHoverValue] = useState(0);
  const displayValue = hoverValue || value;

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (readOnly || !onChange) return;

      if (e.key === "ArrowRight" || e.key === "ArrowUp") {
        e.preventDefault();
        onChange(Math.min(5, value + 1));
      } else if (e.key === "ArrowLeft" || e.key === "ArrowDown") {
        e.preventDefault();
        onChange(Math.max(1, value - 1));
      }
    },
    [readOnly, onChange, value],
  );

  return (
    <div
      role={readOnly ? "img" : "radiogroup"}
      aria-label={label}
      className={`inline-flex gap-0.5 ${className}`}
      onKeyDown={readOnly ? undefined : handleKeyDown}
    >
      {[1, 2, 3, 4, 5].map((star) => (
        <button
          key={star}
          type="button"
          role={readOnly ? undefined : "radio"}
          aria-checked={readOnly ? undefined : star === value}
          aria-label={readOnly ? undefined : `${String(star)} star${star !== 1 ? "s" : ""}`}
          tabIndex={readOnly ? -1 : star === value ? 0 : -1}
          disabled={readOnly}
          className={`p-0.5 transition-colors ${
            readOnly
              ? "cursor-default"
              : "cursor-pointer hover:scale-110 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          } ${star <= displayValue ? "text-warning" : "text-surface-container-high"}`}
          onClick={readOnly ? undefined : () => onChange?.(star)}
          onMouseEnter={readOnly ? undefined : () => setHoverValue(star)}
          onMouseLeave={readOnly ? undefined : () => setHoverValue(0)}
        >
          <Star
            size={20}
            fill={star <= displayValue ? "currentColor" : "none"}
            strokeWidth={star <= displayValue ? 0 : 1.5}
          />
        </button>
      ))}
    </div>
  );
}
