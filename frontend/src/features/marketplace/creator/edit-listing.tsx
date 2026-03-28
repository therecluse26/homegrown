import { useState, useEffect, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, Upload } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Skeleton,
  Badge,
  FormField,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useListingDetail,
  useUpdateListing,
  useSubmitListing,
  usePublishListing,
  useArchiveListing,
} from "@/hooks/use-marketplace";
import type { UpdateListingCommand } from "@/hooks/use-marketplace";

export function EditListing() {
  const intl = useIntl();
  const { id } = useParams<{ id: string }>();
  const { data: listing, isPending } = useListingDetail(id);
  const updateListing = useUpdateListing(id ?? "");
  const submitListing = useSubmitListing();
  const publishListing = usePublishListing();
  const archiveListing = useArchiveListing();
  const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);

  const [form, setForm] = useState<Partial<UpdateListingCommand>>({});

  useEffect(() => {
    if (listing) {
      setForm({
        title: listing.title,
        description: listing.description,
        price_cents: listing.price_cents,
      });
    }
  }, [listing]);

  const updateField = <K extends keyof UpdateListingCommand>(
    key: K,
    value: UpdateListingCommand[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSave = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      updateListing.mutate(form as UpdateListingCommand);
    },
    [form, updateListing],
  );

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!listing) return null;

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage(
          { id: "marketplace.creator.editListing" },
          { title: listing.title },
        )}
      />

      <RouterLink
        to="/creator"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.creator.dashboard" />
      </RouterLink>

      {/* Status & lifecycle actions */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="type-label-md text-on-surface-variant">
              <FormattedMessage id="marketplace.listing.status" />
            </span>
            <Badge
              variant={listing.status === "published" ? "primary" : "secondary"}
            >
              {listing.status}
            </Badge>
            <span className="type-label-sm text-on-surface-variant">
              v{listing.version}
            </span>
          </div>
          <div className="flex gap-2">
            {listing.status === "draft" && (
              <Button
                variant="primary"
                size="sm"
                onClick={() => submitListing.mutate(listing.id)}
                disabled={submitListing.isPending}
              >
                <Icon icon={Upload} size="sm" className="mr-1" />
                <FormattedMessage id="marketplace.listing.submit" />
              </Button>
            )}
            {listing.status === "submitted" && (
              <Button
                variant="primary"
                size="sm"
                onClick={() => publishListing.mutate(listing.id)}
                disabled={publishListing.isPending}
              >
                <FormattedMessage id="marketplace.listing.publish" />
              </Button>
            )}
            {listing.status === "published" && (
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowArchiveConfirm(true)}
              >
                <FormattedMessage id="marketplace.listing.archive" />
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* Edit form */}
      <Card className="p-card-padding">
        <form onSubmit={handleSave} className="space-y-4">
          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.title",
            })}
          >
            {({ id: fieldId }) => (
              <Input
                id={fieldId}
                value={form.title ?? ""}
                onChange={(e) => updateField("title", e.target.value)}
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.description",
            })}
          >
            {({ id: fieldId }) => (
              <textarea
                id={fieldId}
                value={form.description ?? ""}
                onChange={(e) => updateField("description", e.target.value)}
                className="w-full min-h-[120px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.price",
            })}
          >
            {({ id: fieldId }) => (
              <Input
                id={fieldId}
                type="number"
                min={0}
                step={0.01}
                value={
                  form.price_cents != null
                    ? (form.price_cents / 100).toFixed(2)
                    : ""
                }
                onChange={(e) =>
                  updateField(
                    "price_cents",
                    Math.round(Number(e.target.value) * 100),
                  )
                }
              />
            )}
          </FormField>

          <FormField
            label={intl.formatMessage({
              id: "marketplace.listing.form.changeSummary",
            })}
          >
            {({ id: fieldId }) => (
              <Input
                id={fieldId}
                value={form.change_summary ?? ""}
                onChange={(e) => updateField("change_summary", e.target.value)}
                placeholder="What changed in this version?"
              />
            )}
          </FormField>

          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="submit"
              variant="primary"
              disabled={updateListing.isPending}
            >
              <FormattedMessage id="common.save" />
            </Button>
          </div>
        </form>
      </Card>

      {/* Archive confirmation */}
      <ConfirmationDialog
        open={showArchiveConfirm}
        onClose={() => setShowArchiveConfirm(false)}
        title={intl.formatMessage({ id: "marketplace.listing.archiveConfirm" })}
        confirmLabel={intl.formatMessage({
          id: "marketplace.listing.archive",
        })}
        destructive
        onConfirm={() => {
          archiveListing.mutate(listing.id, {
            onSuccess: () => setShowArchiveConfirm(false),
          });
        }}
        loading={archiveListing.isPending}
      >
        <FormattedMessage id="marketplace.listing.archiveWarning" />
      </ConfirmationDialog>
    </div>
  );
}
