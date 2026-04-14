import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate, useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, Calendar, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  FormField,
  Skeleton,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import {
  useCreateScheduleItem,
  useUpdateScheduleItem,
  useDeleteScheduleItem,
  useScheduleItem,
} from "@/hooks/use-planning";
import type {
  CreateScheduleItemInput,
  ScheduleCategory,
} from "@/hooks/use-planning";
import { useStudents } from "@/hooks/use-family";

const CATEGORIES: ScheduleCategory[] = [
  "lesson",
  "reading",
  "activity",
  "assessment",
  "field_trip",
  "co_op",
  "break",
  "custom",
];

export function ScheduleEditor() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { itemId } = useParams<{ itemId: string }>();
  const isEditing = !!itemId;

  const { data: existingItem, isPending: loadingItem } =
    useScheduleItem(itemId);
  const { data: students } = useStudents();
  const createItem = useCreateScheduleItem();
  const updateItem = useUpdateScheduleItem(itemId ?? "");
  const deleteItem = useDeleteScheduleItem();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const [form, setForm] = useState<Partial<CreateScheduleItemInput>>(() => {
    if (existingItem) {
      return {
        title: existingItem.title,
        description: existingItem.description,
        student_id: existingItem.student_id,
        start_date: existingItem.start_date?.slice(0, 10),
        start_time: existingItem.start_time,
        end_time: existingItem.end_time,
        duration_minutes: existingItem.duration_minutes,
        category: existingItem.category,
        color: existingItem.color,
        notes: existingItem.notes,
      };
    }
    return {
      category: "lesson",
      start_date: new Date().toISOString().slice(0, 10),
    };
  });

  // Sync form when existing item loads (for edit mode)
  const [synced, setSynced] = useState(false);
  if (isEditing && existingItem && !synced) {
    setForm({
      title: existingItem.title,
      description: existingItem.description,
      student_id: existingItem.student_id,
      start_date: existingItem.start_date?.slice(0, 10),
      start_time: existingItem.start_time,
      end_time: existingItem.end_time,
      duration_minutes: existingItem.duration_minutes,
      category: existingItem.category,
      color: existingItem.color,
      notes: existingItem.notes,
    });
    setSynced(true);
  }

  const updateField = <K extends keyof CreateScheduleItemInput>(
    key: K,
    value: CreateScheduleItemInput[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!form.title || !form.start_date) return;

      // Convert HTML date input ("YYYY-MM-DD") → RFC3339 ISO string for Go time.Time [H8]
      const data: CreateScheduleItemInput = {
        title: form.title,
        description: form.description,
        student_id: form.student_id,
        start_date: new Date(form.start_date + "T00:00:00").toISOString(),
        start_time: form.start_time,
        end_time: form.end_time,
        duration_minutes: form.duration_minutes,
        category: form.category,
        color: form.color,
        notes: form.notes,
      };

      if (isEditing) {
        updateItem.mutate(data, {
          onSuccess: () => navigate("/calendar"),
        });
      } else {
        createItem.mutate(data, {
          onSuccess: () => navigate("/calendar"),
        });
      }
    },
    [form, isEditing, createItem, updateItem, navigate],
  );

  if (isEditing && loadingItem) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-radius-md" />
      </div>
    );
  }

  if (isEditing && !loadingItem && !existingItem) {
    return <ResourceNotFound backTo="/calendar" />;
  }

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({
          id: isEditing
            ? "planning.schedule.editTitle"
            : "planning.schedule.pageTitle",
        })}
      />

      <RouterLink
        to="/calendar"
        className="inline-flex items-center gap-1 mb-6 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="planning.schedule.backToCalendar" />
      </RouterLink>

      <Card className="p-card-padding">
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Title */}
          <FormField
            label={intl.formatMessage({
              id: "planning.schedule.form.title",
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

          {/* Description */}
          <FormField
            label={intl.formatMessage({
              id: "planning.schedule.form.description",
            })}
          >
            {({ id }) => (
              <textarea
                id={id}
                value={form.description ?? ""}
                onChange={(e) => updateField("description", e.target.value)}
                rows={2}
                className="w-full min-h-[60px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          {/* Student */}
          {students && students.length > 0 && (
            <FormField
              label={intl.formatMessage({
                id: "planning.schedule.form.student",
              })}
            >
              {({ id }) => (
                <Select
                  id={id}
                  value={form.student_id ?? ""}
                  onChange={(e) =>
                    updateField(
                      "student_id",
                      e.target.value || undefined,
                    )
                  }
                >
                  <option value="">
                    {intl.formatMessage({
                      id: "planning.schedule.form.noStudent",
                    })}
                  </option>
                  {students.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.display_name}
                    </option>
                  ))}
                </Select>
              )}
            </FormField>
          )}

          {/* Date + Time */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <FormField
              label={intl.formatMessage({
                id: "planning.schedule.form.date",
              })}
              required
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="date"
                  value={form.start_date ?? ""}
                  onChange={(e) =>
                    updateField("start_date", e.target.value)
                  }
                  required
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({
                id: "planning.schedule.form.startTime",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="time"
                  value={form.start_time ?? ""}
                  onChange={(e) =>
                    updateField("start_time", e.target.value || undefined)
                  }
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({
                id: "planning.schedule.form.endTime",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="time"
                  value={form.end_time ?? ""}
                  onChange={(e) =>
                    updateField("end_time", e.target.value || undefined)
                  }
                />
              )}
            </FormField>
          </div>

          {/* Duration */}
          <FormField
            label={intl.formatMessage({
              id: "planning.schedule.form.duration",
            })}
          >
            {({ id }) => (
              <Input
                id={id}
                type="number"
                min={1}
                value={form.duration_minutes ?? ""}
                onChange={(e) =>
                  updateField(
                    "duration_minutes",
                    e.target.value ? Number(e.target.value) : undefined,
                  )
                }
              />
            )}
          </FormField>

          {/* Category */}
          <FormField
            label={intl.formatMessage({
              id: "planning.schedule.form.category",
            })}
          >
            {({ id }) => (
              <Select
                id={id}
                value={form.category ?? "custom"}
                onChange={(e) =>
                  updateField(
                    "category",
                    e.target.value as ScheduleCategory,
                  )
                }
              >
                {CATEGORIES.map((cat) => (
                  <option key={cat} value={cat}>
                    {intl.formatMessage({
                      id: `planning.schedule.category.${cat}`,
                    })}
                  </option>
                ))}
              </Select>
            )}
          </FormField>

          {/* Notes */}
          <FormField
            label={intl.formatMessage({
              id: "planning.schedule.form.notes",
            })}
          >
            {({ id }) => (
              <textarea
                id={id}
                value={form.notes ?? ""}
                onChange={(e) => updateField("notes", e.target.value)}
                rows={2}
                className="w-full min-h-[60px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          {/* Actions */}
          <div className="flex items-center justify-between pt-2">
            <div>
              {isEditing && (
                <Button
                  type="button"
                  variant="tertiary"
                  size="sm"
                  onClick={() => setShowDeleteConfirm(true)}
                  className="text-error"
                >
                  <Icon icon={Trash2} size="sm" className="mr-1" />
                  <FormattedMessage id="common.delete" />
                </Button>
              )}
            </div>
            <div className="flex gap-3">
              <Button
                type="button"
                variant="tertiary"
                onClick={() => navigate("/calendar")}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={
                  !form.title ||
                  !form.start_date ||
                  createItem.isPending ||
                  updateItem.isPending
                }
              >
                <Icon icon={Calendar} size="sm" className="mr-1" />
                <FormattedMessage id="planning.schedule.form.submit" />
              </Button>
            </div>
          </div>
        </form>
      </Card>

      {/* Delete confirmation */}
      {isEditing && (
        <ConfirmationDialog
          open={showDeleteConfirm}
          onClose={() => setShowDeleteConfirm(false)}
          title={intl.formatMessage({ id: "common.delete" })}
          confirmLabel={intl.formatMessage({ id: "common.delete" })}
          destructive
          onConfirm={() =>
            deleteItem.mutate(itemId, {
              onSuccess: () => navigate("/calendar"),
            })
          }
          loading={deleteItem.isPending}
        >
          <FormattedMessage id="planning.schedule.form.deleteConfirm" />
        </ConfirmationDialog>
      )}
    </div>
  );
}
