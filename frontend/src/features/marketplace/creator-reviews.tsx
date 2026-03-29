import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { MessageSquare, Star } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useCreatorReviews,
  useRespondToReview,
  type CreatorReview,
} from "@/hooks/use-marketplace";

function StarRating({ rating }: { rating: number }) {
  const intl = useIntl();
  return (
    <div
      className="flex items-center gap-0.5"
      aria-label={intl.formatMessage(
        { id: "marketplace.reviews.ratingLabel" },
        { rating },
      )}
      role="img"
    >
      {Array.from({ length: 5 }).map((_, i) => (
        <Icon
          key={i}
          icon={Star}
          size="xs"
          aria-hidden
          className={
            i < rating
              ? "text-tertiary fill-tertiary"
              : "text-outline-variant"
          }
        />
      ))}
    </div>
  );
}

function ReviewCard({ review }: { review: CreatorReview }) {
  const intl = useIntl();
  const respondMutation = useRespondToReview();
  const [isResponding, setIsResponding] = useState(false);
  const [responseText, setResponseText] = useState(
    review.response ?? "",
  );

  async function handleRespond(e: React.FormEvent) {
    e.preventDefault();
    if (!responseText.trim()) return;
    await respondMutation.mutateAsync({
      reviewId: review.id,
      response: responseText.trim(),
    });
    setIsResponding(false);
  }

  return (
    <Card className="flex flex-col gap-3">
      {/* Listing title */}
      <p className="type-label-sm text-on-surface-variant">
        <FormattedMessage
          id="marketplace.reviews.forListing"
          values={{ title: review.listing_title }}
        />
      </p>

      {/* Reviewer and rating */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="type-body-sm text-on-surface font-medium">
            {review.reviewer_name}
          </span>
          <StarRating rating={review.rating} />
        </div>
        <span className="type-label-sm text-on-surface-variant">
          {new Date(review.created_at).toLocaleDateString()}
        </span>
      </div>

      {/* Review text */}
      {review.text && (
        <p className="type-body-sm text-on-surface">{review.text}</p>
      )}

      {/* Existing response */}
      {review.response && !isResponding && (
        <div className="rounded-radius-sm bg-surface-container-low px-3 py-2 border-l-2 border-primary">
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage id="marketplace.reviews.yourResponse" />
          </p>
          <p className="type-body-sm text-on-surface">{review.response}</p>
          <Button
            variant="tertiary"
            size="sm"
            className="mt-2"
            onClick={() => setIsResponding(true)}
          >
            <FormattedMessage id="marketplace.reviews.editResponse" />
          </Button>
        </div>
      )}

      {/* Respond / edit form */}
      {!review.response && !isResponding && (
        <Button
          variant="tertiary"
          size="sm"
          className="self-start"
          onClick={() => setIsResponding(true)}
        >
          <Icon icon={MessageSquare} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="marketplace.reviews.respond" />
        </Button>
      )}

      {isResponding && (
        <form
          onSubmit={handleRespond}
          className="flex flex-col gap-2"
          aria-label={intl.formatMessage({
            id: "marketplace.reviews.responseForm.label",
          })}
        >
          <label
            htmlFor={`response-${review.id}`}
            className="type-label-sm text-on-surface-variant"
          >
            <FormattedMessage id="marketplace.reviews.response.label" />
          </label>
          <textarea
            id={`response-${review.id}`}
            value={responseText}
            onChange={(e) => setResponseText(e.target.value)}
            rows={3}
            className="w-full rounded-radius-sm border border-outline bg-surface px-3 py-2 type-body-sm text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary resize-y"
            placeholder={intl.formatMessage({
              id: "marketplace.reviews.response.placeholder",
            })}
          />
          {respondMutation.error && (
            <p
              role="alert"
              className="type-body-sm text-error"
              aria-live="assertive"
            >
              <FormattedMessage id="error.generic" />
            </p>
          )}
          <div className="flex items-center gap-2">
            <Button
              type="submit"
              variant="primary"
              size="sm"
              loading={respondMutation.isPending}
              disabled={respondMutation.isPending || !responseText.trim()}
            >
              <FormattedMessage id="marketplace.reviews.response.submit" />
            </Button>
            <Button
              type="button"
              variant="tertiary"
              size="sm"
              onClick={() => {
                setIsResponding(false);
                setResponseText(review.response ?? "");
              }}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
          </div>
        </form>
      )}
    </Card>
  );
}

export function CreatorReviews() {
  const intl = useIntl();
  const reviewsQuery = useCreatorReviews();

  if (reviewsQuery.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton className="h-8 w-64 mb-2" />
        <Skeleton className="h-4 w-80 mb-6" />
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-32 rounded-radius-md" />
          ))}
        </div>
      </div>
    );
  }

  if (reviewsQuery.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageTitle
          title={intl.formatMessage({ id: "marketplace.reviews.creator.title" })}
          className="mb-6"
        />
        <Card className="rounded-radius-md bg-error-container p-card-padding">
          <p className="type-body-sm text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const reviews = reviewsQuery.data ?? [];

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.reviews.creator.title" })}
        subtitle={intl.formatMessage(
          { id: "marketplace.reviews.creator.subtitle" },
          { count: reviews.length },
        )}
        className="mb-6"
      />

      {reviews.length === 0 ? (
        <div className="rounded-radius-md bg-surface-container-low px-4 py-8 text-center">
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage id="marketplace.reviews.creator.empty" />
          </p>
          <p className="type-body-sm text-on-surface-variant mt-1">
            <FormattedMessage id="marketplace.reviews.creator.empty.description" />
          </p>
        </div>
      ) : (
        <ul
          className="flex flex-col gap-4"
          role="list"
          aria-label={intl.formatMessage({
            id: "marketplace.reviews.creator.list.label",
          })}
        >
          {reviews.map((review) => (
            <li key={review.id}>
              <ReviewCard review={review} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
