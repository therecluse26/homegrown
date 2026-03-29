import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, ShoppingCart, Star, Download, FileText, Package } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useListingDetail,
  useListingReviews,
  useAddToCart,
  useCreateReview,
  usePurchaseBundle,
} from "@/hooks/use-marketplace";
import type { ReviewResponse } from "@/hooks/use-marketplace";

// ─── Review card ─────────────────────────────────────────────────────────────

function ReviewCard({ review }: { review: ReviewResponse }) {
  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-2 mb-2">
        <div className="flex gap-0.5">
          {[1, 2, 3, 4, 5].map((star) => (
            <Icon
              key={star}
              icon={Star}
              size="xs"
              className={
                star <= review.rating ? "text-warning" : "text-on-surface-variant/30"
              }
            />
          ))}
        </div>
        <span className="type-label-sm text-on-surface-variant">
          {review.reviewer_name ?? "Anonymous"}
        </span>
        <span className="type-label-sm text-on-surface-variant">
          {new Date(review.created_at).toLocaleDateString()}
        </span>
      </div>
      {review.review_text && (
        <p className="type-body-sm text-on-surface">{review.review_text}</p>
      )}
      {review.creator_response && (
        <div className="mt-3 pl-4 border-l-2 border-primary/30">
          <p className="type-label-sm text-primary mb-1">Creator response</p>
          <p className="type-body-sm text-on-surface-variant">
            {review.creator_response}
          </p>
        </div>
      )}
    </Card>
  );
}

// ─── Listing detail page ─────────────────────────────────────────────────────

export function ListingDetail() {
  const intl = useIntl();
  const { id } = useParams<{ id: string }>();
  const { data: listing, isPending } = useListingDetail(id);
  const { data: reviewsResp } = useListingReviews(id);
  const reviews = reviewsResp?.data;
  const addToCart = useAddToCart();
  const purchaseBundle = usePurchaseBundle();
  const createReview = useCreateReview(id ?? "");

  const [reviewRating, setReviewRating] = useState(5);
  const [reviewText, setReviewText] = useState("");
  const [showReviewForm, setShowReviewForm] = useState(false);

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-radius-md" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!listing) return null;

  const price =
    listing.price_cents === 0
      ? "Free"
      : `$${(listing.price_cents / 100).toFixed(2)}`;

  const handleSubmitReview = (e: React.FormEvent) => {
    e.preventDefault();
    createReview.mutate(
      { rating: reviewRating, review_text: reviewText || undefined },
      {
        onSuccess: () => {
          setShowReviewForm(false);
          setReviewText("");
          setReviewRating(5);
        },
      },
    );
  };

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={listing.title} />

      <RouterLink
        to="/marketplace"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.title" />
      </RouterLink>

      {/* Main listing card */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-start gap-6">
          {listing.thumbnail_url ? (
            <div className="w-48 h-48 rounded-radius-md overflow-hidden shrink-0 bg-surface-container-low">
              <img
                src={listing.thumbnail_url}
                alt={listing.title}
                className="w-full h-full object-cover"
              />
            </div>
          ) : (
            <div className="w-48 h-48 rounded-radius-md shrink-0 bg-surface-container-low flex items-center justify-center">
              <Icon
                icon={FileText}
                size="xl"
                className="text-on-surface-variant"
              />
            </div>
          )}

          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <Badge variant="secondary">{listing.content_type.replace("_", " ")}</Badge>
              <Badge variant="default">{listing.status}</Badge>
              {listing.is_bundle && (
                <Badge variant="primary">
                  <Icon icon={Package} size="xs" className="mr-1" />
                  <FormattedMessage id="marketplace.bundle.badge" />
                </Badge>
              )}
            </div>

            <h1 className="type-headline-sm text-on-surface mb-2">
              {listing.title}
            </h1>

            <p className="type-label-md text-on-surface-variant mb-3">
              {listing.publisher_name}
            </p>

            {listing.rating_count > 0 && (
              <div className="flex items-center gap-2 mb-3">
                <div className="flex gap-0.5">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <Icon
                      key={star}
                      icon={Star}
                      size="sm"
                      className={
                        star <= Math.round(listing.rating_avg)
                          ? "text-warning"
                          : "text-on-surface-variant/30"
                      }
                    />
                  ))}
                </div>
                <span className="type-label-md text-on-surface-variant">
                  {listing.rating_avg.toFixed(1)} ({listing.rating_count}{" "}
                  <FormattedMessage id="marketplace.reviews" />)
                </span>
              </div>
            )}

            <p className="type-headline-md text-primary mb-4">{price}</p>

            <div className="flex gap-2 flex-wrap">
              {listing.is_bundle && listing.bundle_id ? (
                <Button
                  variant="primary"
                  onClick={() => purchaseBundle.mutate(listing.bundle_id!)}
                  disabled={purchaseBundle.isPending}
                >
                  <Icon icon={Package} size="sm" className="mr-1" />
                  <FormattedMessage
                    id="marketplace.bundle.buyBundle"
                    values={{ price: `$${(listing.price_cents / 100).toFixed(2)}` }}
                  />
                </Button>
              ) : listing.price_cents > 0 ? (
                <Button
                  variant="primary"
                  onClick={() => addToCart.mutate(listing.id)}
                  disabled={addToCart.isPending}
                >
                  <Icon icon={ShoppingCart} size="sm" className="mr-1" />
                  <FormattedMessage id="marketplace.addToCart" />
                </Button>
              ) : (
                <Button variant="primary">
                  <Icon icon={Download} size="sm" className="mr-1" />
                  <FormattedMessage id="marketplace.getFree" />
                </Button>
              )}
            </div>
          </div>
        </div>
      </Card>

      {/* Description */}
      <Card className="p-card-padding mb-6">
        <h2 className="type-title-md text-on-surface mb-3">
          <FormattedMessage id="marketplace.description" />
        </h2>
        <p className="type-body-md text-on-surface whitespace-pre-wrap">
          {listing.description}
        </p>

        {/* Tags */}
        <div className="flex flex-wrap gap-2 mt-4">
          {listing.subject_tags.map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
          {listing.worldview_tags.map((tag) => (
            <Badge key={tag} variant="default">
              {tag}
            </Badge>
          ))}
        </div>

        {(listing.grade_min != null || listing.grade_max != null) && (
          <p className="type-label-md text-on-surface-variant mt-3">
            <FormattedMessage
              id="marketplace.gradeRange"
              values={{
                min: listing.grade_min ?? "K",
                max: listing.grade_max ?? "12",
              }}
            />
          </p>
        )}
      </Card>

      {/* Bundle contents */}
      {listing.is_bundle && listing.bundle_items && listing.bundle_items.length > 0 && (
        <Card className="p-card-padding mb-6">
          <h2 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="marketplace.bundle.contents" />
          </h2>
          <div className="space-y-2">
            {listing.bundle_items.map((item) => (
              <div
                key={item.listing_id}
                className="flex items-center gap-3 p-2 rounded-radius-sm bg-surface-container-low"
              >
                <Icon icon={FileText} size="sm" className="text-on-surface-variant shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="type-body-sm text-on-surface truncate">{item.title}</p>
                  <p className="type-label-sm text-on-surface-variant">
                    {item.content_type.replace("_", " ")}
                  </p>
                </div>
                <span className="type-label-sm text-on-surface-variant shrink-0">
                  {item.price_cents === 0
                    ? intl.formatMessage({ id: "price.free" })
                    : `$${(item.price_cents / 100).toFixed(2)}`}
                </span>
              </div>
            ))}
          </div>
          <p className="type-label-sm text-on-surface-variant mt-3">
            <FormattedMessage
              id="marketplace.bundle.itemCount"
              values={{ count: listing.bundle_items.length }}
            />
          </p>
        </Card>
      )}

      {/* Files */}
      {listing.files.length > 0 && (
        <Card className="p-card-padding mb-6">
          <h2 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="marketplace.files" />
          </h2>
          <div className="space-y-2">
            {listing.files.map((file) => (
              <div
                key={file.id}
                className="flex items-center gap-3 p-2 rounded-radius-sm bg-surface-container-low"
              >
                <Icon icon={FileText} size="sm" className="text-on-surface-variant" />
                <div className="flex-1 min-w-0">
                  <p className="type-body-sm text-on-surface truncate">
                    {file.file_name}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {(file.file_size_bytes / 1024 / 1024).toFixed(1)} MB
                  </p>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Reviews */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="type-title-md text-on-surface">
            <FormattedMessage id="marketplace.reviews" /> ({reviews?.length ?? 0})
          </h2>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowReviewForm(!showReviewForm)}
          >
            <FormattedMessage id="marketplace.writeReview" />
          </Button>
        </div>

        {showReviewForm && (
          <Card className="p-card-padding mb-4">
            <form onSubmit={handleSubmitReview} className="space-y-3">
              <div>
                <label className="type-label-md text-on-surface block mb-1">
                  <FormattedMessage id="marketplace.review.rating" />
                </label>
                <div className="flex gap-1">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <button
                      key={star}
                      type="button"
                      onClick={() => setReviewRating(star)}
                      className="p-0.5"
                    >
                      <Icon
                        icon={Star}
                        size="md"
                        className={
                          star <= reviewRating
                            ? "text-warning"
                            : "text-on-surface-variant/30"
                        }
                      />
                    </button>
                  ))}
                </div>
              </div>
              <textarea
                value={reviewText}
                onChange={(e) => setReviewText(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "marketplace.review.placeholder",
                })}
                className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="tertiary"
                  size="sm"
                  onClick={() => setShowReviewForm(false)}
                >
                  <FormattedMessage id="common.cancel" />
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  size="sm"
                  disabled={createReview.isPending}
                >
                  <FormattedMessage id="marketplace.review.submit" />
                </Button>
              </div>
            </form>
          </Card>
        )}

        <div className="space-y-3">
          {reviews?.map((review) => (
            <ReviewCard key={review.id} review={review} />
          ))}
        </div>
      </div>
    </div>
  );
}
