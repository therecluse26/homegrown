import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft, Plus, Trash2, LayoutTemplate, CheckCircle2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  FormField,
  Skeleton,
  Badge,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useScheduleTemplates,
  useCreateScheduleTemplate,
  useApplyScheduleTemplate,
  useDeleteScheduleTemplate,
} from "@/hooks/use-planning";
import type {
  ScheduleTemplate,
  ScheduleTemplateItem,
  ScheduleCategory,
} from "@/hooks/use-planning";

// ─── Day grid helper ─────────────────────────────────────────────────────────

const DAYS_OF_WEEK = [
  { value: 1, labelId: "social.events.recurrence.day.mon", abbr: "M" },
  { value: 2, labelId: "social.events.recurrence.day.tue", abbr: "T" },
  { value: 3, labelId: "social.events.recurrence.day.wed", abbr: "W" },
  { value: 4, labelId: "social.events.recurrence.day.thu", abbr: "Th" },
  { value: 5, labelId: "social.events.recurrence.day.fri", abbr: "F" },
  { value: 6, labelId: "social.events.recurrence.day.sat", abbr: "Sa" },
  { value: 0, labelId: "social.events.recurrence.day.sun", abbr: "Su" },
];

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

// ─── Template card ───────────────────────────────────────────────────────────

function TemplateCard({
  template,
  onApply,
  onDelete,
}: {
  template: ScheduleTemplate;
  onApply: (template: ScheduleTemplate) => void;
  onDelete: (templateId: string) => void;
}) {
  const intl = useIntl();

  return (
    <Card className="p-card-padding">
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <h2 className="type-title-sm text-on-surface">{template.name}</h2>
            {template.is_default && (
              <Badge variant="secondary">
                <FormattedMessage id="planning.templates.badge.default" />
              </Badge>
            )}
            {template.methodology_slug && (
              <Badge variant="primary">{template.methodology_slug}</Badge>
            )}
          </div>
          {template.description && (
            <p className="type-body-sm text-on-surface-variant line-clamp-2">
              {template.description}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => onApply(template)}
          >
            <Icon icon={CheckCircle2} size="sm" className="mr-1" />
            <FormattedMessage id="planning.templates.apply" />
          </Button>
          {!template.is_default && (
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => onDelete(template.id)}
              aria-label={intl.formatMessage(
                { id: "planning.templates.delete.label" },
                { name: template.name },
              )}
            >
              <Icon icon={Trash2} size="sm" className="text-error" />
            </Button>
          )}
        </div>
      </div>

      {/* Item count and day distribution */}
      <p className="type-label-sm text-on-surface-variant mb-2">
        <FormattedMessage
          id="planning.templates.itemCount"
          values={{ count: template.items.length }}
        />
      </p>

      {/* Day grid preview */}
      {template.items.length > 0 && (
        <div className="flex gap-1 flex-wrap">
          {DAYS_OF_WEEK.map(({ value, abbr }) => {
            const hasItems = template.items.some((i) => i.day_of_week === value);
            return (
              <span
                key={value}
                className={`w-7 h-7 rounded-full flex items-center justify-center type-label-sm ${
                  hasItems
                    ? "bg-primary text-on-primary"
                    : "bg-surface-container-high text-on-surface-variant"
                }`}
              >
                {abbr}
              </span>
            );
          })}
        </div>
      )}
    </Card>
  );
}

// ─── Create template form ────────────────────────────────────────────────────

interface NewTemplateItemRow {
  title: string;
  category: ScheduleCategory;
  day_of_week: number;
  start_time: string;
  duration_minutes: number;
}

function emptyItemRow(): NewTemplateItemRow {
  return {
    title: "",
    category: "lesson",
    day_of_week: 1,
    start_time: "09:00",
    duration_minutes: 30,
  };
}

function CreateTemplateForm({ onSuccess }: { onSuccess: () => void }) {
  const intl = useIntl();
  const createTemplate = useCreateScheduleTemplate();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [items, setItems] = useState<NewTemplateItemRow[]>([emptyItemRow()]);

  const updateItem = (
    index: number,
    field: keyof NewTemplateItemRow,
    value: string | number,
  ) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [field]: value } as NewTemplateItemRow;
      return next;
    });
  };

  const addItem = () => setItems((prev) => [...prev, emptyItemRow()]);
  const removeItem = (index: number) =>
    setItems((prev) => prev.filter((_, i) => i !== index));

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!name.trim()) return;

      const templateItems: ScheduleTemplateItem[] = items
        .filter((i) => i.title.trim())
        .map((i) => ({
          title: i.title,
          category: i.category,
          day_of_week: i.day_of_week,
          start_time: i.start_time,
          duration_minutes: i.duration_minutes,
        }));

      createTemplate.mutate(
        { name, description: description || undefined, items: templateItems },
        { onSuccess },
      );
    },
    [name, description, items, createTemplate, onSuccess],
  );

  return (
    <Card className="p-card-padding">
      <h2 className="type-title-md text-on-surface mb-4">
        <FormattedMessage id="planning.templates.create.title" />
      </h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <FormField
          label={intl.formatMessage({ id: "planning.templates.form.name" })}
          required
        >
          {({ id }) => (
            <Input
              id={id}
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "planning.templates.form.description" })}
        >
          {({ id }) => (
            <textarea
              id={id}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="w-full min-h-[60px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          )}
        </FormField>

        {/* Weekly pattern items */}
        <div>
          <p className="type-label-lg text-on-surface mb-2">
            <FormattedMessage id="planning.templates.form.items" />
          </p>
          <div className="space-y-3">
            {items.map((item, index) => (
              <div
                key={index}
                className="p-3 rounded-radius-md bg-surface-container-low space-y-2"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <FormField
                    label={intl.formatMessage({ id: "planning.templates.form.item.title" })}
                  >
                    {({ id }) => (
                      <Input
                        id={id}
                        value={item.title}
                        onChange={(e) => updateItem(index, "title", e.target.value)}
                        placeholder={intl.formatMessage({
                          id: "planning.templates.form.item.title.placeholder",
                        })}
                      />
                    )}
                  </FormField>
                  <FormField
                    label={intl.formatMessage({ id: "planning.templates.form.item.category" })}
                  >
                    {({ id }) => (
                      <select
                        id={id}
                        value={item.category}
                        onChange={(e) =>
                          updateItem(index, "category", e.target.value)
                        }
                        className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                      >
                        {CATEGORIES.map((cat) => (
                          <option key={cat} value={cat}>
                            {intl.formatMessage({
                              id: `planning.schedule.category.${cat}`,
                            })}
                          </option>
                        ))}
                      </select>
                    )}
                  </FormField>
                </div>
                <div className="grid grid-cols-3 gap-2">
                  <FormField
                    label={intl.formatMessage({ id: "planning.templates.form.item.day" })}
                  >
                    {({ id }) => (
                      <select
                        id={id}
                        value={item.day_of_week}
                        onChange={(e) =>
                          updateItem(index, "day_of_week", Number(e.target.value))
                        }
                        className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                      >
                        {DAYS_OF_WEEK.map(({ value, labelId }) => (
                          <option key={value} value={value}>
                            {intl.formatMessage({ id: labelId })}
                          </option>
                        ))}
                      </select>
                    )}
                  </FormField>
                  <FormField
                    label={intl.formatMessage({ id: "planning.templates.form.item.startTime" })}
                  >
                    {({ id }) => (
                      <Input
                        id={id}
                        type="time"
                        value={item.start_time}
                        onChange={(e) =>
                          updateItem(index, "start_time", e.target.value)
                        }
                      />
                    )}
                  </FormField>
                  <FormField
                    label={intl.formatMessage({ id: "planning.templates.form.item.duration" })}
                  >
                    {({ id }) => (
                      <Input
                        id={id}
                        type="number"
                        min={1}
                        value={item.duration_minutes}
                        onChange={(e) =>
                          updateItem(
                            index,
                            "duration_minutes",
                            Number(e.target.value),
                          )
                        }
                      />
                    )}
                  </FormField>
                </div>
                {items.length > 1 && (
                  <div className="flex justify-end">
                    <button
                      type="button"
                      onClick={() => removeItem(index)}
                      className="type-label-sm text-error hover:underline"
                    >
                      <FormattedMessage id="planning.templates.form.item.remove" />
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
          <Button
            type="button"
            variant="tertiary"
            size="sm"
            onClick={addItem}
            className="mt-2"
          >
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="planning.templates.form.addItem" />
          </Button>
        </div>

        <div className="flex justify-end gap-3 pt-2">
          <Button type="submit" variant="primary" disabled={!name.trim() || createTemplate.isPending}>
            <Icon icon={LayoutTemplate} size="sm" className="mr-1" />
            <FormattedMessage id="planning.templates.create.submit" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Apply confirmation dialog ───────────────────────────────────────────────

function ApplyTemplateDialog({
  template,
  onClose,
}: {
  template: ScheduleTemplate | null;
  onClose: () => void;
}) {
  const intl = useIntl();
  const applyTemplate = useApplyScheduleTemplate();
  const today = new Date().toISOString().slice(0, 10);

  // find Monday of the current week
  const weekStart = (() => {
    const d = new Date();
    const day = d.getDay();
    const diff = day === 0 ? -6 : 1 - day;
    d.setDate(d.getDate() + diff);
    return d.toISOString().slice(0, 10);
  })();

  const handleConfirm = useCallback(() => {
    if (!template) return;
    applyTemplate.mutate(
      { templateId: template.id, week_start_date: weekStart },
      { onSuccess: onClose },
    );
  }, [template, applyTemplate, weekStart, onClose]);

  return (
    <ConfirmationDialog
      open={!!template}
      onClose={onClose}
      title={intl.formatMessage({ id: "planning.templates.apply.title" })}
      confirmLabel={intl.formatMessage({ id: "planning.templates.apply.confirm" })}
      onConfirm={handleConfirm}
      loading={applyTemplate.isPending}
    >
      {template && (
        <FormattedMessage
          id="planning.templates.apply.description"
          values={{
            count: template.items.length,
            name: template.name,
            date: today,
          }}
        />
      )}
    </ConfirmationDialog>
  );
}

// ─── Page ────────────────────────────────────────────────────────────────────

export function ScheduleTemplates() {
  const intl = useIntl();

  const { data: templates, isPending, isError } = useScheduleTemplates();
  const deleteTemplate = useDeleteScheduleTemplate();

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [applyTarget, setApplyTarget] = useState<ScheduleTemplate | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const handleDeleteConfirm = useCallback(() => {
    if (!deleteTarget) return;
    deleteTemplate.mutate(deleteTarget, {
      onSuccess: () => setDeleteTarget(null),
    });
  }, [deleteTarget, deleteTemplate]);

  const defaultTemplates = templates?.filter((t) => t.is_default) ?? [];
  const customTemplates = templates?.filter((t) => !t.is_default) ?? [];

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "planning.templates.pageTitle" })}
      />

      <RouterLink
        to="/calendar"
        className="inline-flex items-center gap-1 mb-6 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="planning.schedule.backToCalendar" />
      </RouterLink>

      <div className="flex items-center justify-between mb-6">
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowCreateForm((v) => !v)}
        >
          <Icon icon={Plus} size="sm" className="mr-1" />
          <FormattedMessage id="planning.templates.create" />
        </Button>
      </div>

      {/* Create form */}
      {showCreateForm && (
        <div className="mb-8">
          <CreateTemplateForm onSuccess={() => setShowCreateForm(false)} />
        </div>
      )}

      {/* Loading state */}
      {isPending && (
        <div className="space-y-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-28 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {/* Error state */}
      {isError && (
        <Card className="p-card-padding bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      )}

      {/* Default templates */}
      {!isPending && !isError && defaultTemplates.length > 0 && (
        <section className="mb-8">
          <h2 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="planning.templates.section.defaults" />
          </h2>
          <div className="space-y-3">
            {defaultTemplates.map((t) => (
              <TemplateCard
                key={t.id}
                template={t}
                onApply={setApplyTarget}
                onDelete={setDeleteTarget}
              />
            ))}
          </div>
        </section>
      )}

      {/* Custom templates */}
      {!isPending && !isError && (
        <section>
          <h2 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="planning.templates.section.custom" />
          </h2>
          {customTemplates.length === 0 ? (
            <Card className="p-card-padding">
              <p className="type-body-md text-on-surface-variant text-center py-4">
                <FormattedMessage id="planning.templates.empty" />
              </p>
            </Card>
          ) : (
            <div className="space-y-3">
              {customTemplates.map((t) => (
                <TemplateCard
                  key={t.id}
                  template={t}
                  onApply={setApplyTarget}
                  onDelete={setDeleteTarget}
                />
              ))}
            </div>
          )}
        </section>
      )}

      {/* Apply confirmation */}
      <ApplyTemplateDialog
        template={applyTarget}
        onClose={() => setApplyTarget(null)}
      />

      {/* Delete confirmation */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title={intl.formatMessage({ id: "planning.templates.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "planning.templates.delete.confirm" })}
        destructive
        onConfirm={handleDeleteConfirm}
        loading={deleteTemplate.isPending}
      >
        <FormattedMessage id="planning.templates.delete.description" />
      </ConfirmationDialog>
    </div>
  );
}
