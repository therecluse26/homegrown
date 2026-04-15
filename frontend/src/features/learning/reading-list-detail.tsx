import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import { ArrowLeft, Edit2, Trash2, BookOpen } from "lucide-react";
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
  useReadingListDetail,
  useUpdateReadingList,
  useDeleteReadingList,
} from "@/hooks/use-reading";

export function ReadingListDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  const { data: list, isPending } = useReadingListDetail(id ?? "");
  const updateList = useUpdateReadingList();
  const deleteList = useDeleteReadingList();

  const [editName, setEditName] = useState("");
  const [editDesc, setEditDesc] = useState("");

  function startEdit() {
    if (!list) return;
    setEditName(list.name);
    setEditDesc(list.description ?? "");
    setEditing(true);
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!list || !editName.trim()) return;
    updateList.mutate(
      {
        id: list.id,
        name: editName.trim(),
        description: editDesc.trim() || undefined,
      },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (!list) return;
    deleteList.mutate(list.id, {
      onSuccess: () => void navigate("/learning/reading-lists"),
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

  if (!list) {
    return <ResourceNotFound backTo="/learning/reading-lists" />;
  }

  const completedCount = list.items.filter(
    (item) => item.progress?.status === "completed",
  ).length;

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle title={list.name} />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/learning/reading-lists"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="readingListDetail.backToLists" />
        </RouterLink>
      </div>

      {editing ? (
        <Card>
          <form onSubmit={handleSave} className="space-y-5">
            <div>
              <label
                htmlFor="edit-name"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="readingListDetail.name" />
              </label>
              <Input
                id="edit-name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                required
              />
            </div>
            <div>
              <label
                htmlFor="edit-desc"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="readingListDetail.description" />
              </label>
              <Textarea
                id="edit-desc"
                value={editDesc}
                onChange={(e) => setEditDesc(e.target.value)}
                rows={3}
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
                loading={updateList.isPending}
                disabled={!editName.trim()}
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
                {list.name}
              </h1>
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

            {list.description && (
              <p className="type-body-sm text-on-surface-variant mb-4">
                {list.description}
              </p>
            )}

            <div className="flex items-center gap-4">
              <div>
                <p className="type-label-sm text-on-surface-variant">
                  <FormattedMessage id="readingListDetail.bookCount" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {list.items.length}
                </p>
              </div>
              <div>
                <p className="type-label-sm text-on-surface-variant">
                  <FormattedMessage id="readingListDetail.completed" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {completedCount} / {list.items.length}
                </p>
              </div>
            </div>
          </Card>

          {/* Book list */}
          <Card className="p-card-padding">
            <div className="flex items-center justify-between mb-3">
              <h3 className="type-title-md text-on-surface">
                <FormattedMessage id="readingListDetail.books" />
              </h3>
              <RouterLink
                to={`/learning/reading-lists/${id}/books`}
                className="type-label-md text-primary hover:underline"
              >
                <FormattedMessage id="readingListDetail.manageBooks" />
              </RouterLink>
            </div>

            {list.items.length === 0 ? (
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="readingListDetail.emptyList" />
              </p>
            ) : (
              <div className="space-y-2">
                {list.items.map((item) => (
                  <div
                    key={item.reading_item.id}
                    className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
                  >
                    <div className="flex items-center gap-3">
                      <Icon
                        icon={BookOpen}
                        size="sm"
                        className="text-on-surface-variant"
                      />
                      <div>
                        <p className="type-body-sm text-on-surface">
                          {item.reading_item.title}
                        </p>
                        {item.reading_item.author && (
                          <p className="type-label-sm text-on-surface-variant">
                            {item.reading_item.author}
                          </p>
                        )}
                      </div>
                    </div>
                    {item.progress && (
                      <Badge
                        variant={
                          item.progress.status === "completed"
                            ? "primary"
                            : "secondary"
                        }
                      >
                        {item.progress.status}
                      </Badge>
                    )}
                  </div>
                ))}
              </div>
            )}
          </Card>
        </>
      )}

      <Modal
        open={showDelete}
        onClose={() => setShowDelete(false)}
        title={intl.formatMessage({ id: "readingListDetail.deleteTitle" })}
      >
        <div className="space-y-4">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="readingListDetail.deleteConfirm" />
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
              loading={deleteList.isPending}
            >
              <FormattedMessage id="common.delete" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
