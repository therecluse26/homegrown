import { useState, useCallback, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate, Link as RouterLink } from "react-router";
import { ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useCreateListing, useCreatorProfile } from "@/hooks/use-marketplace";
import type { CreateListingCommand } from "@/hooks/use-marketplace";

const CONTENT_TYPES = [
  "curriculum",
  "worksheet",
  "unit_study",
  "video",
  "book_list",
  "assessment",
  "lesson_plan",
  "printable",
  "project_guide",
  "reading_guide",
  "course",
  "interactive_quiz",
  "lesson_sequence",
] as const;

export function CreateListing() {
  const intl = useIntl();
  const navigate = useNavigate();
  const createListing = useCreateListing();
  const { data: creatorProfile } = useCreatorProfile();

  const [form, setForm] = useState<Partial<CreateListingCommand>>({
    price_cents: 0,
    content_type: "curriculum",
    methodology_tags: [],
    subject_tags: [],
  });

  // Auto-fill publisher_id from creator profile
  useEffect(() => {
    if (creatorProfile?.id && !form.publisher_id) {
      setForm((prev) => ({ ...prev, publisher_id: creatorProfile.id }));
    }
  }, [creatorProfile?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const [subjectInput, setSubjectInput] = useState("");

  const updateField = <K extends keyof CreateListingCommand>(
    key: K,
    value: CreateListingCommand[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const addSubjectTag = () => {
    const tag = subjectInput.trim();
    if (tag && !form.subject_tags?.includes(tag)) {
      updateField("subject_tags", [...(form.subject_tags ?? []), tag]);
      setSubjectInput("");
    }
  };

  const removeSubjectTag = (tag: string) => {
    updateField(
      "subject_tags",
      (form.subject_tags ?? []).filter((t) => t !== tag),
    );
  };

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!form.title || !form.description || !form.publisher_id) return;

      createListing.mutate(form as CreateListingCommand, {
        onSuccess: (listing) => {
          navigate(`/creator/listings/${listing.id}/edit`);
        },
      });
    },
    [form, createListing, navigate],
  );

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.creator.createListing" })}
      />

      <RouterLink
        to="/creator"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.creator.dashboard" />
      </RouterLink>

      <Card className="p-card-padding">
        <form onSubmit={handleSubmit} className="space-y-4">
          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.title",
            })}
            required
          >
            {({ id }) => (
              <Input
                id={id}
                value={form.title ?? ""}
                onChange={(e) => updateField("title", e.target.value)}
                required
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.description",
            })}
            required
          >
            {({ id }) => (
              <textarea
                id={id}
                value={form.description ?? ""}
                onChange={(e) => updateField("description", e.target.value)}
                required
                className="w-full min-h-[120px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.price",
            })}
            required
          >
            {({ id }) => (
              <Input
                id={id}
                type="number"
                min={0}
                step={0.01}
                value={
                  form.price_cents != null
                    ? (form.price_cents / 100).toFixed(2)
                    : ""
                }
                onChange={(e) =>
                  updateField(
                    "price_cents",
                    Math.round(Number(e.target.value) * 100),
                  )
                }
                required
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.contentType",
            })}
            required
          >
            {({ id }) => (
              <select
                id={id}
                value={form.content_type ?? ""}
                onChange={(e) => updateField("content_type", e.target.value)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              >
                {CONTENT_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type.replace(/_/g, " ")}
                  </option>
                ))}
              </select>
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.publisherId",
            })}
            required
          >
            {({ id }) => (
              <div>
                <p
                  id={id}
                  className="type-body-md text-on-surface rounded-radius-sm bg-surface-container px-3 py-2"
                >
                  {creatorProfile?.store_name ?? form.publisher_id ?? "—"}
                </p>
                <input type="hidden" name="publisher_id" value={form.publisher_id ?? ""} />
              </div>
            )}
          </FormField>

          {/* Subject tags */}
          <div>
            <label className="type-label-md text-on-surface block mb-1">
              <FormattedMessage id="marketplace.listing.form.subjects" />
            </label>
            <div className="flex gap-2 mb-2">
              <Input
                value={subjectInput}
                onChange={(e) => setSubjectInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addSubjectTag();
                  }
                }}
                placeholder="Add subject tag"
                className="flex-1"
              />
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={addSubjectTag}
              >
                Add
              </Button>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {form.subject_tags?.map((tag) => (
                <button
                  key={tag}
                  type="button"
                  onClick={() => removeSubjectTag(tag)}
                  className="px-2 py-1 rounded-radius-sm bg-secondary-container text-on-secondary-container type-label-sm hover:opacity-80"
                >
                  {tag} ×
                </button>
              ))}
            </div>
          </div>

          {/* Grade range */}
          <div className="grid grid-cols-2 gap-4">
            <FormField
              label={intl.formatMessage({
                id: "marketplace.listing.form.gradeMin",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="number"
                  min={0}
                  max={12}
                  value={form.grade_min ?? ""}
                  onChange={(e) =>
                    updateField(
                      "grade_min",
                      e.target.value ? Number(e.target.value) : undefined,
                    )
                  }
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({
                id: "marketplace.listing.form.gradeMax",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="number"
                  min={0}
                  max={12}
                  value={form.grade_max ?? ""}
                  onChange={(e) =>
                    updateField(
                      "grade_max",
                      e.target.value ? Number(e.target.value) : undefined,
                    )
                  }
                />
              )}
            </FormField>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <RouterLink to="/creator">
              <Button type="button" variant="tertiary">
                <FormattedMessage id="common.cancel" />
              </Button>
            </RouterLink>
            <Button
              type="submit"
              variant="primary"
              disabled={
                !form.title ||
                !form.description ||
                !form.publisher_id ||
                createListing.isPending
              }
            >
              <FormattedMessage id="marketplace.creator.createListing" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
