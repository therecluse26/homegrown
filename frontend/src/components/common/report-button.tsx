import { useState, useCallback } from "react";
import { Flag } from "lucide-react";
import { Modal } from "../ui/modal";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { Textarea } from "../ui/textarea";
import { FormField } from "../ui/form-field";
import { Icon } from "../ui/icon";

type ReportCategory =
  | "inappropriate_content"
  | "harassment"
  | "spam"
  | "misinformation"
  | "child_safety"
  | "methodology_hostility"
  | "other";

const CATEGORIES: { value: ReportCategory; label: string }[] = [
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "harassment", label: "Harassment" },
  { value: "spam", label: "Spam" },
  { value: "misinformation", label: "Misinformation" },
  { value: "child_safety", label: "Child safety concern" },
  { value: "methodology_hostility", label: "Methodology hostility" },
  { value: "other", label: "Other" },
];

type ReportButtonProps = {
  /** Type of entity being reported */
  targetType: string;
  /** ID of the entity being reported */
  targetId: string;
  /** Called when report is submitted */
  onSubmit?: (data: {
    targetType: string;
    targetId: string;
    category: ReportCategory;
    description: string;
  }) => void;
  className?: string;
};

export function ReportButton({
  targetType,
  targetId,
  onSubmit,
  className = "",
}: ReportButtonProps) {
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState<ReportCategory | "">("");
  const [description, setDescription] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = useCallback(() => {
    if (!category) return;

    onSubmit?.({
      targetType,
      targetId,
      category,
      description,
    });

    setSubmitted(true);
  }, [category, description, targetType, targetId, onSubmit]);

  const handleClose = useCallback(() => {
    setOpen(false);
    // Reset after animation
    setTimeout(() => {
      setCategory("");
      setDescription("");
      setSubmitted(false);
    }, 200);
  }, []);

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={`inline-flex items-center gap-1.5 text-on-surface-variant type-body-sm hover:text-error transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${className}`}
        aria-label="Report"
      >
        <Icon icon={Flag} size="sm" />
        <span>Report</span>
      </button>

      <Modal
        open={open}
        onClose={handleClose}
        title="Report content"
      >
        {submitted ? (
          <div className="flex flex-col items-center gap-4 py-4 text-center">
            <p className="type-title-md text-on-surface">
              Thank you for your report
            </p>
            <p className="type-body-md text-on-surface-variant">
              Our team will review this and take appropriate action.
            </p>
            <Button variant="primary" onClick={handleClose}>
              Done
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <h2 className="type-title-lg text-on-surface">Report content</h2>

            <FormField label="Category" required>
              {({ id }) => (
                <Select
                  id={id}
                  value={category}
                  onChange={(e) =>
                    setCategory(e.target.value as ReportCategory | "")
                  }
                >
                  <option value="" disabled>
                    Select a reason
                  </option>
                  {CATEGORIES.map((cat) => (
                    <option key={cat.value} value={cat.value}>
                      {cat.label}
                    </option>
                  ))}
                </Select>
              )}
            </FormField>

            <FormField label="Description" hint="Optional — provide additional details">
              {({ id }) => (
                <Textarea
                  id={id}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Describe the issue..."
                  rows={3}
                />
              )}
            </FormField>

            <p className="type-body-sm text-on-surface-variant">
              Please review our{" "}
              <a href="/legal/guidelines" className="text-primary underline">
                Community Guidelines
              </a>{" "}
              for more information.
            </p>

            <div className="flex justify-end gap-3">
              <Button variant="tertiary" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={handleSubmit}
                disabled={!category}
              >
                Submit Report
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </>
  );
}
