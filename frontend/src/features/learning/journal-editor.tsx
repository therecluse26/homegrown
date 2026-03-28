import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
  Textarea,
} from "@/components/ui";
import { SubjectPicker } from "@/components/common/subject-picker";
import { useStudents } from "@/hooks/use-family";
import {
  useCreateJournalEntry,
  type JournalEntryType,
} from "@/hooks/use-journals";

export function JournalEditor() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [entryType, setEntryType] = useState<JournalEntryType>("freeform");
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [entryDate, setEntryDate] = useState(
    new Date().toISOString().slice(0, 10),
  );

  const effectiveStudent =
    studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const createEntry = useCreateJournalEntry(effectiveStudent);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !content.trim()) return;
    createEntry.mutate(
      {
        entry_type: entryType,
        title: title.trim() || undefined,
        content: content.trim(),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        entry_date: entryDate || undefined,
      },
      {
        onSuccess: () => {
          void navigate("/learning/journals");
        },
      },
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning/journals")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="journalEditor.title" />
        </h1>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          <div>
            <label
              htmlFor="journal-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="journal-student"
                value={effectiveStudent}
                onChange={(e) => setStudentId(e.target.value)}
                required
              >
                <option value="">
                  {intl.formatMessage({
                    id: "activityLog.selectStudent",
                  })}
                </option>
                {students?.map((s) => (
                  <option key={s.id} value={s.id ?? ""}>
                    {s.display_name}
                  </option>
                ))}
              </Select>
            )}
          </div>

          {/* Entry type */}
          <div>
            <label
              htmlFor="entry-type"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journalEditor.entryType" />
            </label>
            <Select
              id="entry-type"
              value={entryType}
              onChange={(e) =>
                setEntryType(e.target.value as JournalEntryType)
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

          {/* Title */}
          <div>
            <label
              htmlFor="entry-title"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journalEditor.entryTitle" />
            </label>
            <Input
              id="entry-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "journalEditor.entryTitle.placeholder",
              })}
            />
          </div>

          {/* Content */}
          <div>
            <label
              htmlFor="entry-content"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journalEditor.content" />
            </label>
            <Textarea
              id="entry-content"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={intl.formatMessage({
                id: "journalEditor.content.placeholder",
              })}
              rows={10}
              required
            />
          </div>

          {/* Date + subjects */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="entry-date"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="journalEditor.date" />
              </label>
              <Input
                id="entry-date"
                type="date"
                value={entryDate}
                onChange={(e) => setEntryDate(e.target.value)}
              />
            </div>
          </div>

          <div>
            <label className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="activityLog.field.subjects" />
            </label>
            <SubjectPicker
              value={subjectTags}
              onChange={setSubjectTags}
              allowCustom
            />
          </div>

          {/* Actions */}
          <div className="flex gap-2 justify-end pt-2">
            <Button
              variant="tertiary"
              size="sm"
              type="button"
              onClick={() => void navigate("/learning/journals")}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              size="sm"
              type="submit"
              loading={createEntry.isPending}
              disabled={!effectiveStudent || !content.trim()}
            >
              <FormattedMessage id="journalEditor.save" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
