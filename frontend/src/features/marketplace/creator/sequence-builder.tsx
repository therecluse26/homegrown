import { useState, useCallback, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink, useParams } from "react-router";
import {
  ArrowLeft,
  Plus,
  Trash2,
  GripVertical,
  ArrowUp,
  ArrowDown,
  ListOrdered,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  FormField,
  Badge,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useSequenceDef } from "@/hooks/use-sequences";

// ─── Types ───────────────────────────────────────────────────────────────────

interface SequenceStep {
  id: string;
  title: string;
  description: string;
  content_type: string;
  content_id: string;
  duration_minutes: number;
}

let nextStepId = 1;
function genStepId() {
  return `step-${nextStepId++}`;
}

const STEP_CONTENT_TYPES = [
  { value: "video", label: "Video" },
  { value: "reading", label: "Reading" },
  { value: "quiz", label: "Quiz" },
  { value: "activity", label: "Activity" },
  { value: "assignment", label: "Assignment" },
] as const;

// ─── Step editor ─────────────────────────────────────────────────────────────

function StepEditor({
  step,
  index,
  total,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
}: {
  step: SequenceStep;
  index: number;
  total: number;
  onChange: (s: SequenceStep) => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}) {
  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-2 mb-3">
        <Icon
          icon={GripVertical}
          size="sm"
          className="text-on-surface-variant cursor-grab"
        />
        <Badge variant="secondary">Step {index + 1}</Badge>
        <span className="type-label-sm text-on-surface-variant flex-1 truncate">
          {step.title || "Untitled step"}
        </span>
        <div className="flex gap-1">
          <button
            onClick={onMoveUp}
            disabled={index === 0}
            className="p-1 rounded-radius-sm hover:bg-surface-container-low disabled:opacity-30"
            aria-label={`Move step ${index + 1} up`}
          >
            <Icon icon={ArrowUp} size="xs" />
          </button>
          <button
            onClick={onMoveDown}
            disabled={index === total - 1}
            className="p-1 rounded-radius-sm hover:bg-surface-container-low disabled:opacity-30"
            aria-label={`Move step ${index + 1} down`}
          >
            <Icon icon={ArrowDown} size="xs" />
          </button>
          <button
            onClick={onRemove}
            className="p-1 rounded-radius-sm text-error hover:bg-error-container/30"
            aria-label="Remove step"
          >
            <Icon icon={Trash2} size="xs" />
          </button>
        </div>
      </div>

      <div className="space-y-3">
        <FormField label="Step title" required>
          {({ id }) => (
            <Input
              id={id}
              value={step.title}
              onChange={(e) => onChange({ ...step, title: e.target.value })}
              required
            />
          )}
        </FormField>

        <FormField label="Description">
          {({ id }) => (
            <textarea
              id={id}
              value={step.description}
              onChange={(e) =>
                onChange({ ...step, description: e.target.value })
              }
              className="w-full min-h-[60px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          )}
        </FormField>

        <div className="grid grid-cols-3 gap-3">
          <FormField label="Content type">
            {({ id }) => (
              <select
                id={id}
                value={step.content_type}
                onChange={(e) =>
                  onChange({ ...step, content_type: e.target.value })
                }
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              >
                {STEP_CONTENT_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            )}
          </FormField>

          <FormField label="Content ID">
            {({ id }) => (
              <Input
                id={id}
                value={step.content_id}
                onChange={(e) =>
                  onChange({ ...step, content_id: e.target.value })
                }
                placeholder="Content UUID"
              />
            )}
          </FormField>

          <FormField label="Duration (min)">
            {({ id }) => (
              <Input
                id={id}
                type="number"
                min={1}
                value={step.duration_minutes}
                onChange={(e) =>
                  onChange({
                    ...step,
                    duration_minutes: Number(e.target.value),
                  })
                }
              />
            )}
          </FormField>
        </div>
      </div>
    </Card>
  );
}

// ─── Sequence builder page ───────────────────────────────────────────────────

export function SequenceBuilder() {
  const intl = useIntl();
  const { id } = useParams<{ id: string }>();
  const { data: seqDef, isPending: loadingDef } = useSequenceDef(id ?? "");
  const isEditing = !!id;

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [steps, setSteps] = useState<SequenceStep[]>([]);
  const [reorderAnnouncement, setReorderAnnouncement] = useState("");
  const [hydrated, setHydrated] = useState(false);

  // Populate form state when existing sequence data loads
  useEffect(() => {
    if (seqDef && !hydrated) {
      setTitle(seqDef.title);
      setDescription(seqDef.description ?? "");
      setSteps(
        (seqDef.items ?? []).map((item) => ({
          id: genStepId(),
          title: `Step ${item.sort_order + 1}`,
          description: "",
          content_type: item.content_type,
          content_id: item.content_id,
          duration_minutes: 15,
        })),
      );
      setHydrated(true);
    }
  }, [seqDef, hydrated]);

  const addStep = useCallback(() => {
    setSteps((prev) => [
      ...prev,
      {
        id: genStepId(),
        title: "",
        description: "",
        content_type: "video",
        content_id: "",
        duration_minutes: 15,
      },
    ]);
  }, []);

  const updateStep = useCallback((index: number, s: SequenceStep) => {
    setSteps((prev) => prev.map((item, i) => (i === index ? s : item)));
  }, []);

  const removeStep = useCallback((index: number) => {
    setSteps((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const moveStep = useCallback((from: number, to: number) => {
    setSteps((prev) => {
      if (to < 0 || to >= prev.length) return prev;
      const arr = [...prev];
      const removed = arr.splice(from, 1);
      if (removed.length === 0) return prev;
      arr.splice(to, 0, removed[0]!);
      setReorderAnnouncement(
        `Step moved from position ${from + 1} to position ${to + 1} of ${prev.length}`,
      );
      return arr;
    });
  }, []);

  const totalDuration = steps.reduce(
    (sum, s) => sum + s.duration_minutes,
    0,
  );

  if (isEditing && loadingDef) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={isEditing
          ? intl.formatMessage({ id: "marketplace.sequence.editBuilder" }, { fallback: "Edit Sequence" })
          : intl.formatMessage({ id: "marketplace.sequence.builder" })}
      />

      <RouterLink
        to="/creator"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.creator.dashboard" />
      </RouterLink>

      {/* Sequence meta */}
      <Card className="p-card-padding mb-6">
        <div className="space-y-3">
          <FormField
            label={intl.formatMessage({
              id: "marketplace.sequence.title",
            })}
            required
          >
            {({ id }) => (
              <Input
                id={id}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Sequence title"
                required
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.sequence.description",
            })}
          >
            {({ id }) => (
              <textarea
                id={id}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          <div className="flex items-center gap-4">
            <span className="type-label-md text-on-surface-variant flex items-center gap-1">
              <Icon icon={ListOrdered} size="xs" />
              {steps.length} steps
            </span>
            <span className="type-label-md text-on-surface-variant">
              {totalDuration} min total
            </span>
          </div>
        </div>
      </Card>

      {/* Screen reader reorder announcements */}
      <div aria-live="assertive" className="sr-only">
        {reorderAnnouncement}
      </div>

      {/* Steps */}
      <div className="space-y-4 mb-6" role="list" aria-label="Sequence steps">
        {steps.map((step, i) => (
          <StepEditor
            key={step.id}
            step={step}
            index={i}
            total={steps.length}
            onChange={(updated) => updateStep(i, updated)}
            onRemove={() => removeStep(i)}
            onMoveUp={() => moveStep(i, i - 1)}
            onMoveDown={() => moveStep(i, i + 1)}
          />
        ))}
      </div>

      {/* Add step */}
      <Button variant="secondary" onClick={addStep} className="w-full mb-6">
        <Icon icon={Plus} size="sm" className="mr-1" />
        <FormattedMessage id="marketplace.sequence.addStep" />
      </Button>

      {/* Save */}
      <div className="flex justify-end">
        <Button variant="primary" disabled={!title || steps.length === 0}>
          <FormattedMessage id="common.save" />
        </Button>
      </div>
    </div>
  );
}
