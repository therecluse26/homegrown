import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import { ArrowLeft, Edit2, Trash2, BookMarked } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Input,
  Textarea,
  Modal,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import {
  useActivityLogEntry,
  useUpdateActivityLog,
  useDeleteActivityLog,
} from "@/hooks/use-activities";

export function TriviumDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  const searchParams = new URLSearchParams(
    typeof window !== "undefined" ? window.location.search : "",
  );
  const studentId = searchParams.get("studentId") ?? "";

  const { data: activity, isPending } = useActivityLogEntry(
    studentId,
    id ?? "",
  );
  const updateActivity = useUpdateActivityLog(
    activity?.student_id ?? studentId,
  );
  const deleteActivity = useDeleteActivityLog(
    activity?.student_id ?? studentId,
  );

  const [editTitle, setEditTitle] = useState("");
  const [editDesc, setEditDesc] = useState("");
  const [editDuration, setEditDuration] = useState("");
  const [editDate, setEditDate] = useState("");

  function startEdit() {
    if (!activity) return;
    setEditTitle(activity.title);
    setEditDesc(activity.description ?? "");
    setEditDuration(
      activity.duration_minutes ? String(activity.duration_minutes) : "",
    );
    setEditDate(activity.activity_date?.slice(0, 10) ?? "");
    setEditing(true);
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!activity || !editTitle.trim()) return;
    updateActivity.mutate(
      {
        id: activity.id,
        title: editTitle.trim(),
        description: editDesc.trim() || undefined,
        duration_minutes: editDuration ? Number(editDuration) : undefined,
        activity_date: editDate ? `${editDate}T00:00:00Z` : undefined,
      },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (!activity) return;
    deleteActivity.mutate(activity.id, {
      onSuccess: () => void navigate("/learning/trivium-tracker"),
    });
  }

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!activity) {
    return <ResourceNotFound backTo="/learning/trivium-tracker" />;
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle title={activity.title} />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/learning/trivium-tracker"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="triviumDetail.back" />
        </RouterLink>
      </div>

      {editing ? (
        <Card>
          <form onSubmit={handleSave} className="space-y-5">
            <div>
              <label
                htmlFor="edit-title"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="triviumDetail.entryTitle" />
              </label>
              <Input
                id="edit-title"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                required
              />
            </div>
            <div>
              <label
                htmlFor="edit-desc"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="triviumDetail.description" />
              </label>
              <Textarea
                id="edit-desc"
                value={editDesc}
                onChange={(e) => setEditDesc(e.target.value)}
                rows={6}
              />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="edit-date"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="activityNew.date" />
                </label>
                <Input
                  id="edit-date"
                  type="date"
                  value={editDate}
                  onChange={(e) => setEditDate(e.target.value)}
                />
              </div>
              <div>
                <label
                  htmlFor="edit-duration"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="activityNew.duration" />
                </label>
                <Input
                  id="edit-duration"
                  type="number"
                  min={0}
                  value={editDuration}
                  onChange={(e) => setEditDuration(e.target.value)}
                />
              </div>
            </div>
            <div className="flex gap-2 justify-end pt-2">
              <Button
                variant="tertiary"
                size="sm"
                type="button"
                onClick={() => setEditing(false)}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                variant="primary"
                size="sm"
                type="submit"
                loading={updateActivity.isPending}
                disabled={!editTitle.trim()}
              >
                <FormattedMessage id="common.save" />
              </Button>
            </div>
          </form>
        </Card>
      ) : (
        <Card className="p-card-padding">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Icon icon={BookMarked} size="md" className="text-primary" />
              <h1 className="type-headline-sm text-on-surface">
                {activity.title}
              </h1>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="tertiary" size="sm" onClick={startEdit}>
                <Icon icon={Edit2} size="sm" />
              </Button>
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowDelete(true)}
              >
                <Icon icon={Trash2} size="sm" />
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <p className="type-label-sm text-on-surface-variant mb-1">
                <FormattedMessage id="triviumDetail.date" />
              </p>
              <p className="type-body-sm text-on-surface">
                {new Date(activity.activity_date).toLocaleDateString()}
              </p>
            </div>
            {activity.duration_minutes != null && (
              <div>
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="triviumDetail.duration" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {activity.duration_minutes}{" "}
                  <FormattedMessage id="activityDetail.minutes" />
                </p>
              </div>
            )}
          </div>

          {activity.subject_tags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-4">
              {activity.subject_tags.map((tag) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ))}
            </div>
          )}

          {activity.description && (
            <div className="type-body-sm text-on-surface whitespace-pre-wrap">
              {activity.description}
            </div>
          )}
        </Card>
      )}

      <Modal
        open={showDelete}
        onClose={() => setShowDelete(false)}
        title={intl.formatMessage({ id: "triviumDetail.deleteTitle" })}
      >
        <div className="space-y-4">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="triviumDetail.deleteConfirm" />
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowDelete(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleDelete}
              loading={deleteActivity.isPending}
            >
              <FormattedMessage id="common.delete" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
