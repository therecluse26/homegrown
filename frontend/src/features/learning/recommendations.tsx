import { useState, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { X, ChevronDown, ChevronUp, Sparkles, MoreHorizontal } from "lucide-react";
import { Badge, Button, Card, Icon, Skeleton } from "@/components/ui";
import {
  useRecommendations,
  useDismissRecommendation,
  useBlockCategory,
  useUndoDismiss,
  type Recommendation,
} from "@/hooks/use-recommendations";

function RecommendationCard({ rec }: { rec: Recommendation }) {
  const intl = useIntl();
  const dismiss = useDismissRecommendation();
  const blockCategory = useBlockCategory();
  const undoDismiss = useUndoDismiss();
  const [reasonExpanded, setReasonExpanded] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [lastDismissedId, setLastDismissedId] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  async function handleDismiss() {
    setDismissed(true);
    setLastDismissedId(rec.id);
    await dismiss.mutateAsync(rec.id);
  }

  async function handleUndo() {
    if (!lastDismissedId) return;
    await undoDismiss.mutateAsync(lastDismissedId);
    setDismissed(false);
    setLastDismissedId(null);
  }

  async function handleBlockCategory() {
    setMenuOpen(false);
    await blockCategory.mutateAsync(rec.category);
  }

  if (dismissed) {
    return (
      <div
        className="flex-shrink-0 w-72 rounded-radius-md bg-surface-container-low px-4 py-6 flex flex-col items-center justify-center gap-3 text-center"
        role="status"
        aria-live="polite"
      >
        <p className="type-body-sm text-on-surface-variant">
          <FormattedMessage id="recommendations.card.dismissed" />
        </p>
        <Button variant="tertiary" size="sm" onClick={handleUndo}>
          <FormattedMessage id="recommendations.card.undo" />
        </Button>
      </div>
    );
  }

  return (
    <Card className="flex-shrink-0 w-72 flex flex-col gap-3 relative">
      {/* Header row */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="type-label-sm text-on-surface-variant bg-surface-container-low px-2 py-0.5 rounded-radius-sm">
            {rec.category}
          </span>
          {rec.ai_generated && (
            <Badge variant="secondary">
              <Icon icon={Sparkles} size="xs" aria-hidden className="mr-0.5" />
              <FormattedMessage id="recommendations.card.aiBadge" />
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-1">
          {/* Menu */}
          <div className="relative" ref={menuRef}>
            <button
              type="button"
              onClick={() => setMenuOpen((v) => !v)}
              aria-label={intl.formatMessage({ id: "recommendations.card.menu.label" })}
              aria-expanded={menuOpen}
              aria-haspopup="menu"
              className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-surface-container focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <Icon icon={MoreHorizontal} size="sm" aria-hidden />
            </button>
            {menuOpen && (
              <div
                role="menu"
                aria-label={intl.formatMessage({ id: "recommendations.card.menu.label" })}
                className="absolute right-0 mt-1 w-44 rounded-radius-sm bg-surface-container shadow-elevation-2 border border-outline-variant z-10"
              >
                <button
                  type="button"
                  role="menuitem"
                  onClick={handleBlockCategory}
                  disabled={blockCategory.isPending}
                  className="w-full text-left px-3 py-2 type-body-sm text-on-surface hover:bg-surface-container-high focus:outline-none focus:bg-surface-container-high"
                >
                  <FormattedMessage
                    id="recommendations.card.blockCategory"
                    values={{ category: rec.category }}
                  />
                </button>
              </div>
            )}
          </div>

          {/* Dismiss */}
          <button
            type="button"
            onClick={handleDismiss}
            disabled={dismiss.isPending}
            aria-label={intl.formatMessage(
              { id: "recommendations.card.dismiss.label" },
              { title: rec.title },
            )}
            className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-surface-container focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <Icon icon={X} size="sm" aria-hidden />
          </button>
        </div>
      </div>

      {/* Title */}
      <div>
        {rec.link ? (
          <a
            href={rec.link}
            target="_blank"
            rel="noopener noreferrer"
            className="type-title-sm text-on-surface font-semibold hover:text-primary focus:outline-none focus:underline"
          >
            {rec.title}
          </a>
        ) : (
          <p className="type-title-sm text-on-surface font-semibold">
            {rec.title}
          </p>
        )}
        <p className="type-body-sm text-on-surface-variant mt-0.5 line-clamp-2">
          {rec.description}
        </p>
      </div>

      {/* Why recommended */}
      <div>
        <button
          type="button"
          onClick={() => setReasonExpanded((v) => !v)}
          aria-expanded={reasonExpanded}
          className="flex items-center gap-1 type-label-sm text-on-surface-variant hover:text-on-surface focus:outline-none focus:underline"
        >
          <FormattedMessage id="recommendations.card.whyRecommended" />
          <Icon
            icon={reasonExpanded ? ChevronUp : ChevronDown}
            size="xs"
            aria-hidden
          />
        </button>
        {reasonExpanded && (
          <p className="mt-1.5 type-body-sm text-on-surface-variant italic">
            {rec.reason}
          </p>
        )}
      </div>
    </Card>
  );
}

export function Recommendations() {
  const intl = useIntl();
  const { data: recommendations, isPending, error } = useRecommendations();

  if (isPending) {
    return (
      <section
        aria-labelledby="recommendations-heading"
        className="py-2"
      >
        <h2
          id="recommendations-heading"
          className="type-title-sm text-on-surface font-semibold mb-3"
        >
          <FormattedMessage id="recommendations.section.title" />
        </h2>
        <div className="flex gap-3 overflow-x-auto pb-2">
          {[1, 2, 3].map((n) => (
            <Skeleton
              key={n}
              className="flex-shrink-0 w-72 h-36 rounded-radius-md"
            />
          ))}
        </div>
      </section>
    );
  }

  if (error || !recommendations || recommendations.length === 0) {
    return null;
  }

  return (
    <section
      aria-labelledby="recommendations-heading"
      className="py-2"
    >
      <h2
        id="recommendations-heading"
        className="type-title-sm text-on-surface font-semibold mb-3"
      >
        <FormattedMessage id="recommendations.section.title" />
      </h2>
      <div
        className="flex gap-3 overflow-x-auto pb-2 -mx-1 px-1"
        role="list"
        aria-label={intl.formatMessage({ id: "recommendations.section.list.label" })}
      >
        {recommendations.map((rec) => (
          <div key={rec.id} role="listitem">
            <RecommendationCard rec={rec} />
          </div>
        ))}
      </div>
    </section>
  );
}
