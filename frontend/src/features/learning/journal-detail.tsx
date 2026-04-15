import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import { ArrowLeft, Edit2, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Input,
  Select,
  Textarea,
  Modal,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import { SubjectPicker } from "@/components/common/subject-picker";
import {
  useJournalEntry,
  useUpdateJournalEntry,
  useDeleteJournalEntry,
  type JournalEntryType,
} from "@/hooks/use-journals";

export function JournalDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  // We need to find the student ID — journal entries are student-scoped.
  // The entry itself contains student_id, so we fetch with a placeholder first,
  // then re-fetch once we know the student. For detail pages accessed from the
  // journal list, we encode studentId in the query string.
  const searchParams = new URLSearchParams(
    typeof window !== "undefined" ? window.location.search : "",
  );
  const studentId = searchParams.get("studentId") ?? "";

  const { data: entry, isPending } = useJournalEntry(studentId, id ?? "");
  const updateEntry = useUpdateJournalEntry(entry?.student_id ?? studentId);
  const deleteEntry = useDeleteJournalEntry(entry?.student_id ?? studentId);

  // Edit form state
  const [editTitle, setEditTitle] = useState("");
  const [editContent, setEditContent] = useState("");
  const [editType, setEditType] = useState<JournalEntryType>("freeform");
  const [editTags, setEditTags] = useState<string[]>([]);
  const [editDate, setEditDate] = useState("");

  function startEdit() {
    if (!entry) return;
    setEditTitle(entry.title ?? "");
    setEditContent(entry.content);
    setEditType(entry.entry_type);
    setEditTags(entry.subject_tags ?? []);
    setEditDate(entry.entry_date?.slice(0, 10) ?? "");
    setEditing(true);
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!entry || !editContent.trim()) return;
    updateEntry.mutate(
      {
        id: entry.id,
        entry_type: editType,
        title: editTitle.trim() || undefined,
        content: editContent.trim(),
        subject_tags: editTags.length > 0 ? editTags : undefined,
        entry_date: editDate ? `${editDate}T00:00:00Z` : undefined,
      },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (!entry) return;
    deleteEntry.mutate(entry.id, {
      onSuccess: () => void navigate("/learning/journals"),
    });
  }

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!entry) {
    return <ResourceNotFound backTo="/learning/journals" />;
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={
          entry.title ??
          intl.formatMessage({ id: "journalDetail.untitled" })
        }
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/learning/journals"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="journalDetail.backToJournals" />
        </RouterLink>
      </div>

      {editing ? (
        <Card>
          <form onSubmit={handleSave} className="space-y-5">
            <div>
              <label
                htmlFor="edit-type"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="journalEditor.entryType" />
              </label>
              <Select
                id="edit-type"
                value={editType}
                onChange={(e) =>
                  setEditType(e.target.value as JournalEntryType)
                }
              >
                <option value="freeform">
                  {intl.formatMessage({ id: "journals.type.freeform" })}
                </option>
                <option value="narration">
                  {intl.formatMessage({ id: "journals.type.narration" })}
                </option>
                <option value="reflection">
                  {intl.formatMessage({ id: "journals.type.reflection" })}
                </option>
              </Select>
            </div>

            <div>
              <label
                htmlFor="edit-title"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="journalEditor.entryTitle" />
              </label>
              <Input
                id="edit-title"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
              />
            </div>

            <div>
              <label
                htmlFor="edit-content"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="journalEditor.content" />
              </label>
              <Textarea
                id="edit-content"
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                rows={10}
                required
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="edit-date"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="journalEditor.date" />
                </label>
                <Input
                  id="edit-date"
                  type="date"
                  value={editDate}
                  onChange={(e) => setEditDate(e.target.value)}
                />
              </div>
            </div>

            <div>
              <label className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="activityLog.field.subjects" />
              </label>
              <SubjectPicker
                value={editTags}
                onChange={setEditTags}
                allowCustom
              />
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
                loading={updateEntry.isPending}
                disabled={!editContent.trim()}
              >
                <FormattedMessage id="common.save" />
              </Button>
            </div>
          </form>
        </Card>
      ) : (
        <>
          <Card className="p-card-padding">
            <div className="flex items-center justify-between mb-4">
              <h1 className="type-headline-sm text-on-surface">
                {entry.title ??
                  intl.formatMessage({ id: "journalDetail.untitled" })}
              </h1>
              <div className="flex items-center gap-2">
                <Badge variant="secondary">{entry.entry_type}</Badge>
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
                  <FormattedMessage id="journalDetail.date" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {new Date(entry.entry_date).toLocaleDateString()}
                </p>
              </div>
              <div>
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="journalDetail.created" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {new Date(entry.created_at).toLocaleDateString()}
                </p>
              </div>
            </div>

            {entry.subject_tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-4">
                {entry.subject_tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}

            <div className="type-body-sm text-on-surface whitespace-pre-wrap">
              {entry.content}
            </div>
          </Card>

          {entry.attachments.length > 0 && (
            <Card className="p-card-padding">
              <h3 className="type-title-md text-on-surface mb-3">
                <FormattedMessage id="journalDetail.attachments" />
              </h3>
              <div className="space-y-2">
                {entry.attachments.map((att, i) => (
                  <a
                    key={i}
                    href={att.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block type-body-sm text-primary hover:underline"
                  >
                    {att.filename ?? att.url}
                  </a>
                ))}
              </div>
            </Card>
          )}
        </>
      )}

      <Modal
        open={showDelete}
        onClose={() => setShowDelete(false)}
        title={intl.formatMessage({ id: "journalDetail.deleteTitle" })}
      >
        <div className="space-y-4">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="journalDetail.deleteConfirm" />
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
              loading={deleteEntry.isPending}
            >
              <FormattedMessage id="common.delete" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
