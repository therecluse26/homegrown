import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, BookOpen, Plus, X } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Input,
  Modal,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import {
  useReadingListDetail,
  useUpdateReadingList,
  useReadingItems,
} from "@/hooks/use-reading";

export function ReadingListBooks() {
  const intl = useIntl();
  const { id } = useParams<{ id: string }>();
  const [showAddModal, setShowAddModal] = useState(false);
  const [bookSearch, setBookSearch] = useState("");

  const { data: list, isPending } = useReadingListDetail(id ?? "");
  const updateList = useUpdateReadingList();
  const { data: searchResults } = useReadingItems({
    search: bookSearch.length >= 2 ? bookSearch : undefined,
  });

  const existingItemIds = new Set(
    list?.items.map((item) => item.reading_item.id) ?? [],
  );

  function handleAddBook(bookId: string) {
    if (!id) return;
    updateList.mutate(
      { id, add_item_ids: [bookId] },
      { onSuccess: () => setShowAddModal(false) },
    );
  }

  function handleRemoveBook(bookId: string) {
    if (!id) return;
    updateList.mutate({ id, remove_item_ids: [bookId] });
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

  const allItems = searchResults?.pages?.flatMap((p) => p.data) ?? [];

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage(
          { id: "readingListBooks.title" },
          { name: list.name },
        )}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to={`/learning/reading-lists/${id}`}
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="readingListBooks.backToList" />
        </RouterLink>
      </div>

      <Card className="p-card-padding">
        <div className="flex items-center justify-between mb-4">
          <h1 className="type-headline-sm text-on-surface">
            <FormattedMessage
              id="readingListBooks.heading"
              values={{ name: list.name }}
            />
          </h1>
          <Button
            variant="primary"
            size="sm"
            onClick={() => setShowAddModal(true)}
          >
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="readingListBooks.addBook" />
          </Button>
        </div>

        {list.items.length === 0 ? (
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="readingListBooks.empty" />
          </p>
        ) : (
          <div className="space-y-2">
            {list.items.map((item) => (
              <div
                key={item.reading_item.id}
                className="flex items-center justify-between py-3 border-b border-outline-variant/10 last:border-0"
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
                    <div className="flex gap-1.5 mt-1">
                      {item.reading_item.subject_tags?.map((tag) => (
                        <Badge key={tag} variant="secondary">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
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
                  <Button
                    variant="tertiary"
                    size="sm"
                    onClick={() => handleRemoveBook(item.reading_item.id)}
                  >
                    <Icon icon={X} size="sm" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Modal
        open={showAddModal}
        onClose={() => {
          setShowAddModal(false);
          setBookSearch("");
        }}
        title={intl.formatMessage({ id: "readingListBooks.addBookTitle" })}
      >
        <div className="space-y-4">
          <Input
            value={bookSearch}
            onChange={(e) => setBookSearch(e.target.value)}
            placeholder={intl.formatMessage({
              id: "readingListBooks.searchPlaceholder",
            })}
          />

          <div className="max-h-64 overflow-y-auto space-y-1">
            {allItems
              .filter((item) => !existingItemIds.has(item.id))
              .map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className="w-full flex items-center justify-between p-2 rounded-radius-sm hover:bg-surface-container-low transition-colors text-left"
                  onClick={() => handleAddBook(item.id)}
                >
                  <div>
                    <p className="type-body-sm text-on-surface">
                      {item.title}
                    </p>
                    {item.author && (
                      <p className="type-label-sm text-on-surface-variant">
                        {item.author}
                      </p>
                    )}
                  </div>
                  <Icon icon={Plus} size="sm" className="text-primary" />
                </button>
              ))}
          </div>
        </div>
      </Modal>
    </div>
  );
}
