import { useState, useMemo, useCallback, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  ArrowUp,
  ArrowDown,
  Plus,
  Trash2,
  Download,
  FileText,
  Eye,
  GripVertical,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
  Badge,
  Input,
  ConfirmationDialog,
  Modal,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import {
  usePortfolioDetail,
  usePortfolioItemCandidates,
  useUpdatePortfolio,
  useGeneratePortfolio,
} from "@/hooks/use-compliance";
import type {
  PortfolioItem,
  PortfolioOrganization,
  PortfolioItemCandidate,
} from "@/hooks/use-compliance";

// ─── Item type labels ──────────────────────────────────────────────────────

const ITEM_TYPE_LABELS: Record<PortfolioItem["item_type"], string> = {
  work_sample: "compliance.portfolio.itemType.workSample",
  assessment: "compliance.portfolio.itemType.assessment",
  attendance: "compliance.portfolio.itemType.attendance",
  journal: "compliance.portfolio.itemType.journal",
  activity: "compliance.portfolio.itemType.activity",
};

// ─── Item row (with keyboard reorder) ──────────────────────────────────────

function ItemRow({
  item,
  index,
  total,
  onMove,
  onRemove,
}: {
  item: PortfolioItem;
  index: number;
  total: number;
  onMove: (from: number, to: number) => void;
  onRemove: (index: number) => void;
}) {
  const intl = useIntl();
  const [grabbed, setGrabbed] = useState(false);
  const liveRef = useRef<HTMLSpanElement>(null);

  const announce = useCallback(
    (msg: string) => {
      if (liveRef.current) liveRef.current.textContent = msg;
    },
    [],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        setGrabbed((prev) => !prev);
        if (!grabbed) {
          announce(
            intl.formatMessage(
              { id: "compliance.portfolio.item.grabbed" },
              { title: item.title, position: index + 1, total },
            ),
          );
        } else {
          announce(
            intl.formatMessage(
              { id: "compliance.portfolio.item.dropped" },
              { title: item.title, position: index + 1, total },
            ),
          );
        }
      } else if (grabbed && e.key === "Escape") {
        e.preventDefault();
        setGrabbed(false);
        announce(
          intl.formatMessage({ id: "compliance.portfolio.item.cancelled" }),
        );
      } else if (grabbed && e.key === "ArrowUp" && index > 0) {
        e.preventDefault();
        onMove(index, index - 1);
        announce(
          intl.formatMessage(
            { id: "compliance.portfolio.item.moved" },
            { title: item.title, position: index, total },
          ),
        );
      } else if (grabbed && e.key === "ArrowDown" && index < total - 1) {
        e.preventDefault();
        onMove(index, index + 1);
        announce(
          intl.formatMessage(
            { id: "compliance.portfolio.item.moved" },
            { title: item.title, position: index + 2, total },
          ),
        );
      }
    },
    [grabbed, index, total, item.title, onMove, intl, announce],
  );

  return (
    <div
      className={`flex items-center gap-3 p-3 rounded-radius-sm transition-colors ${
        grabbed
          ? "bg-primary-container/20 ring-2 ring-primary"
          : "bg-surface-container-low hover:bg-surface-container-high"
      }`}
    >
      <button
        className="p-1 rounded-radius-sm text-on-surface-variant hover:text-on-surface cursor-grab focus:outline-none focus:ring-2 focus:ring-primary touch-target"
        onKeyDown={handleKeyDown}
        aria-label={intl.formatMessage(
          { id: "compliance.portfolio.item.reorder" },
          { title: item.title },
        )}
        aria-roledescription={intl.formatMessage({
          id: "compliance.portfolio.item.draggable",
        })}
      >
        <Icon icon={GripVertical} size="sm" />
      </button>
      <span aria-live="assertive" className="sr-only" ref={liveRef} />

      <div className="flex-1 min-w-0">
        <p className="type-body-md text-on-surface truncate">{item.title}</p>
        <div className="flex items-center gap-2 mt-0.5">
          <Badge variant="secondary">
            <FormattedMessage id={ITEM_TYPE_LABELS[item.item_type]} />
          </Badge>
          {item.subject && (
            <span className="type-label-sm text-on-surface-variant">
              {item.subject}
            </span>
          )}
          <span className="type-label-sm text-on-surface-variant">
            {item.date}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-1 shrink-0">
        <button
          onClick={() => onMove(index, index - 1)}
          disabled={index === 0}
          className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-highest disabled:opacity-30 transition-colors touch-target"
          aria-label={intl.formatMessage({
            id: "compliance.portfolio.item.moveUp",
          })}
        >
          <Icon icon={ArrowUp} size="xs" />
        </button>
        <button
          onClick={() => onMove(index, index + 1)}
          disabled={index === total - 1}
          className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-highest disabled:opacity-30 transition-colors touch-target"
          aria-label={intl.formatMessage({
            id: "compliance.portfolio.item.moveDown",
          })}
        >
          <Icon icon={ArrowDown} size="xs" />
        </button>
        <button
          onClick={() => onRemove(index)}
          className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-error-container hover:text-on-error-container transition-colors touch-target"
          aria-label={intl.formatMessage(
            { id: "compliance.portfolio.item.remove" },
            { title: item.title },
          )}
        >
          <Icon icon={Trash2} size="xs" />
        </button>
      </div>
    </div>
  );
}

// ─── Candidate picker modal ────────────────────────────────────────────────

function CandidatePicker({
  open,
  onClose,
  candidates,
  isPending,
  typeFilter,
  onTypeFilter,
  onAdd,
  existingIds,
}: {
  open: boolean;
  onClose: () => void;
  candidates: PortfolioItemCandidate[] | undefined;
  isPending: boolean;
  typeFilter: PortfolioItem["item_type"] | "";
  onTypeFilter: (t: PortfolioItem["item_type"] | "") => void;
  onAdd: (candidate: PortfolioItemCandidate) => void;
  existingIds: Set<string>;
}) {
  const intl = useIntl();

  if (!open) return null;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={intl.formatMessage({ id: "compliance.portfolio.addItems.title" })}
    >
      <div className="mb-4">
        <Select
          value={typeFilter}
          onChange={(e) =>
            onTypeFilter(e.target.value as PortfolioItem["item_type"] | "")
          }
          aria-label={intl.formatMessage({
            id: "compliance.portfolio.addItems.filterType",
          })}
        >
          <option value="">
            {intl.formatMessage({
              id: "compliance.portfolio.addItems.allTypes",
            })}
          </option>
          {(
            Object.keys(ITEM_TYPE_LABELS) as PortfolioItem["item_type"][]
          ).map((t) => (
            <option key={t} value={t}>
              {intl.formatMessage({ id: ITEM_TYPE_LABELS[t] })}
            </option>
          ))}
        </Select>
      </div>

      {isPending ? (
        <div className="space-y-2">
          {[1, 2, 3, 4].map((n) => (
            <Skeleton key={n} className="h-12 w-full rounded-radius-sm" />
          ))}
        </div>
      ) : !candidates || candidates.length === 0 ? (
        <p className="type-body-md text-on-surface-variant text-center py-8">
          <FormattedMessage id="compliance.portfolio.addItems.empty" />
        </p>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {candidates.map((c) => {
            const alreadyAdded = existingIds.has(c.id);
            return (
              <div
                key={c.id}
                className="flex items-center justify-between gap-3 p-2 rounded-radius-sm bg-surface-container-low"
              >
                <div className="flex-1 min-w-0">
                  <p className="type-body-sm text-on-surface truncate">
                    {c.title}
                  </p>
                  <div className="flex items-center gap-2 mt-0.5">
                    <Badge variant="secondary">
                      <FormattedMessage id={ITEM_TYPE_LABELS[c.item_type]} />
                    </Badge>
                    {c.subject && (
                      <span className="type-label-sm text-on-surface-variant">
                        {c.subject}
                      </span>
                    )}
                  </div>
                </div>
                <Button
                  variant={alreadyAdded ? "tertiary" : "secondary"}
                  size="sm"
                  disabled={alreadyAdded}
                  onClick={() => onAdd(c)}
                >
                  {alreadyAdded ? (
                    <FormattedMessage id="compliance.portfolio.addItems.added" />
                  ) : (
                    <>
                      <Icon icon={Plus} size="xs" className="mr-1" />
                      <FormattedMessage id="compliance.portfolio.addItems.add" />
                    </>
                  )}
                </Button>
              </div>
            );
          })}
        </div>
      )}
    </Modal>
  );
}

// ─── Preview modal ─────────────────────────────────────────────────────────

function PreviewModal({
  open,
  onClose,
  items,
  title,
  studentName,
  dateRange,
}: {
  open: boolean;
  onClose: () => void;
  items: PortfolioItem[];
  title: string;
  studentName: string;
  dateRange: string;
}) {
  const intl = useIntl();

  if (!open) return null;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={intl.formatMessage({
        id: "compliance.portfolio.preview.title",
      })}
    >
      <div className="print:block">
        <div className="mb-4 pb-4 border-b border-outline-variant/20">
          <h3 className="type-headline-sm text-on-surface font-bold">
            {title}
          </h3>
          <p className="type-body-md text-on-surface-variant mt-1">
            {studentName} — {dateRange}
          </p>
        </div>

        {items.length === 0 ? (
          <p className="type-body-md text-on-surface-variant text-center py-8">
            <FormattedMessage id="compliance.portfolio.preview.empty" />
          </p>
        ) : (
          <div className="space-y-3">
            {items.map((item, i) => (
              <div
                key={item.id}
                className="flex items-start gap-3 p-3 bg-surface-container-low rounded-radius-sm"
              >
                <span className="type-label-sm text-on-surface-variant shrink-0 w-6 text-right">
                  {i + 1}.
                </span>
                <div className="flex-1 min-w-0">
                  <p className="type-body-md text-on-surface">{item.title}</p>
                  <div className="flex items-center gap-2 mt-0.5">
                    <Badge variant="secondary">
                      <FormattedMessage id={ITEM_TYPE_LABELS[item.item_type]} />
                    </Badge>
                    {item.subject && (
                      <span className="type-label-sm text-on-surface-variant">
                        {item.subject}
                      </span>
                    )}
                    <span className="type-label-sm text-on-surface-variant">
                      {item.date}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </Modal>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function PortfolioBuilder() {
  const intl = useIntl();
  const { id, studentId: routeStudentId } = useParams<{ id: string; studentId: string }>();
  const { tier } = useAuth();
  const studentId = routeStudentId ?? "";

  const { data: portfolio, isPending } = usePortfolioDetail(studentId, id);
  const updatePortfolio = useUpdatePortfolio(studentId, id ?? "");
  const generatePortfolio = useGeneratePortfolio(studentId, id ?? "");

  // Local state for item management
  const [items, setItems] = useState<PortfolioItem[]>([]);
  const [itemsDirty, setItemsDirty] = useState(false);
  const [coverName, setCoverName] = useState("");
  const [coverDateRange, setCoverDateRange] = useState("");
  const [organization, setOrganization] =
    useState<PortfolioOrganization>("chronological");

  // Candidate picker
  const [showPicker, setShowPicker] = useState(false);
  const [typeFilter, setTypeFilter] = useState<
    PortfolioItem["item_type"] | ""
  >("");

  // Preview and generate
  const [showPreview, setShowPreview] = useState(false);
  const [showGenerateConfirm, setShowGenerateConfirm] = useState(false);

  // Sync portfolio data to local state on load
  const lastSyncedId = useRef<string | null>(null);
  if (portfolio && portfolio.id !== lastSyncedId.current) {
    lastSyncedId.current = portfolio.id;
    setItems(portfolio.items);
    setCoverName(portfolio.cover_student_name ?? portfolio.student_name);
    setCoverDateRange(
      portfolio.cover_date_range ??
        `${portfolio.date_range_start} – ${portfolio.date_range_end}`,
    );
    setOrganization(portfolio.organization);
    setItemsDirty(false);
  }

  const candidateParams = useMemo(
    () =>
      portfolio
        ? {
            student_id: portfolio.student_id,
            date_range_start: portfolio.date_range_start,
            date_range_end: portfolio.date_range_end,
            item_type: typeFilter || undefined,
          }
        : {
            student_id: "",
            date_range_start: "",
            date_range_end: "",
          },
    [portfolio, typeFilter],
  );

  const { data: candidates, isPending: candidatesPending } =
    usePortfolioItemCandidates(candidateParams);

  const existingIds = useMemo(
    () => new Set(items.map((i) => i.source_id)),
    [items],
  );

  const dateRangeLabel = useMemo(() => {
    if (!portfolio) return "";
    const start = new Date(portfolio.date_range_start + "T12:00:00");
    const end = new Date(portfolio.date_range_end + "T12:00:00");
    return `${start.toLocaleDateString(intl.locale, {
      month: "short",
      day: "numeric",
    })} – ${end.toLocaleDateString(intl.locale, {
      month: "short",
      day: "numeric",
      year: "numeric",
    })}`;
  }, [portfolio, intl.locale]);

  // Item management
  const moveItem = useCallback((from: number, to: number) => {
    if (to < 0) return;
    setItems((prev) => {
      if (to >= prev.length) return prev;
      const next = [...prev];
      const [moved] = next.splice(from, 1);
      if (moved) next.splice(to, 0, moved);
      return next;
    });
    setItemsDirty(true);
  }, []);

  const removeItem = useCallback((index: number) => {
    setItems((prev) => prev.filter((_, i) => i !== index));
    setItemsDirty(true);
  }, []);

  const addCandidate = useCallback((c: PortfolioItemCandidate) => {
    setItems((prev) => [
      ...prev,
      {
        id: crypto.randomUUID(),
        item_type: c.item_type,
        title: c.title,
        subject: c.subject,
        date: c.date,
        source_id: c.id,
        sort_order: prev.length,
      },
    ]);
    setItemsDirty(true);
  }, []);

  const handleSave = useCallback(() => {
    updatePortfolio.mutate({
      organization,
      cover_student_name: coverName || undefined,
      cover_date_range: coverDateRange || undefined,
      items: items.map((item, i) => ({
        source_id: item.source_id,
        item_type: item.item_type,
        sort_order: i,
      })),
    });
    setItemsDirty(false);
  }, [items, organization, coverName, coverDateRange, updatePortfolio]);

  const handleGenerate = useCallback(() => {
    generatePortfolio.mutate(undefined, {
      onSuccess: () => setShowGenerateConfirm(false),
    });
  }, [generatePortfolio]);

  if (tier === "free") {
    return <TierGate featureName="Portfolio Builder" />;
  }

  if (isPending) {
    return (
      <div className="max-w-content mx-auto">
        <Skeleton className="h-8 w-48 rounded-radius-sm mb-4" />
        <Skeleton className="h-40 w-full rounded-radius-md mb-4" />
        <Skeleton className="h-60 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!portfolio) {
    return (
      <div className="max-w-content mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "compliance.portfolio.notFound" })}
        />
        <Card className="p-card-padding text-center">
          <p className="type-body-md text-on-surface-variant py-8">
            <FormattedMessage id="compliance.portfolio.notFound" />
          </p>
          <RouterLink to="/compliance/portfolios">
            <Button variant="primary" size="sm">
              <FormattedMessage id="compliance.portfolio.backToList" />
            </Button>
          </RouterLink>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage(
          { id: "compliance.portfolio.builder.pageTitle" },
          { name: portfolio.title },
        )}
      />

      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <RouterLink to="/compliance/portfolios">
          <Button variant="tertiary" size="sm">
            <Icon icon={ArrowLeft} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.portfolio.backToList" />
          </Button>
        </RouterLink>
      </div>

      {/* Cover page config */}
      <Card className="p-card-padding mb-6">
        <h2 className="type-title-sm text-on-surface mb-4">
          <FormattedMessage id="compliance.portfolio.cover.title" />
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label
              htmlFor="cover-name"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.cover.studentName" />
            </label>
            <Input
              id="cover-name"
              value={coverName}
              onChange={(e) => {
                setCoverName(e.target.value);
                setItemsDirty(true);
              }}
            />
          </div>
          <div>
            <label
              htmlFor="cover-dates"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.cover.dateRange" />
            </label>
            <Input
              id="cover-dates"
              value={coverDateRange}
              onChange={(e) => {
                setCoverDateRange(e.target.value);
                setItemsDirty(true);
              }}
            />
          </div>
          <div>
            <label
              htmlFor="cover-org"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.form.organization" />
            </label>
            <Select
              id="cover-org"
              value={organization}
              onChange={(e) => {
                setOrganization(e.target.value as PortfolioOrganization);
                setItemsDirty(true);
              }}
            >
              <option value="chronological">
                {intl.formatMessage({
                  id: "compliance.portfolio.org.chronological",
                })}
              </option>
              <option value="by_subject">
                {intl.formatMessage({
                  id: "compliance.portfolio.org.bySubject",
                })}
              </option>
              <option value="by_type">
                {intl.formatMessage({
                  id: "compliance.portfolio.org.byType",
                })}
              </option>
            </Select>
          </div>
        </div>
      </Card>

      {/* Items section */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="type-title-sm text-on-surface">
            <FormattedMessage id="compliance.portfolio.items.title" />
            <Badge variant="secondary" className="ml-2">
              {items.length}
            </Badge>
          </h2>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowPicker(true)}
          >
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.portfolio.addItems.button" />
          </Button>
        </div>

        {items.length === 0 ? (
          <p className="type-body-md text-on-surface-variant text-center py-8">
            <FormattedMessage id="compliance.portfolio.items.empty" />
          </p>
        ) : (
          <div className="space-y-2" role="list">
            {items.map((item, i) => (
              <div role="listitem" key={item.id}>
                <ItemRow
                  item={item}
                  index={i}
                  total={items.length}
                  onMove={moveItem}
                  onRemove={removeItem}
                />
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Action bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setShowPreview(true)}
            disabled={items.length === 0}
          >
            <Icon icon={Eye} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.portfolio.preview" />
          </Button>
          {portfolio.download_url && (
            <a
              href={portfolio.download_url}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="tertiary" size="sm">
                <Icon icon={Download} size="sm" className="mr-1" />
                <FormattedMessage id="compliance.portfolio.download" />
              </Button>
            </a>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={handleSave}
            disabled={!itemsDirty || updatePortfolio.isPending}
          >
            <FormattedMessage id="common.save" />
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={() => setShowGenerateConfirm(true)}
            disabled={items.length === 0 || generatePortfolio.isPending}
          >
            <Icon icon={FileText} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.portfolio.generate" />
          </Button>
        </div>
      </div>

      {/* Candidate picker */}
      <CandidatePicker
        open={showPicker}
        onClose={() => setShowPicker(false)}
        candidates={candidates}
        isPending={candidatesPending}
        typeFilter={typeFilter}
        onTypeFilter={setTypeFilter}
        onAdd={addCandidate}
        existingIds={existingIds}
      />

      {/* Preview */}
      <PreviewModal
        open={showPreview}
        onClose={() => setShowPreview(false)}
        items={items}
        title={portfolio.title}
        studentName={coverName || portfolio.student_name}
        dateRange={coverDateRange || dateRangeLabel}
      />

      {/* Generate confirmation */}
      <ConfirmationDialog
        open={showGenerateConfirm}
        onConfirm={handleGenerate}
        onClose={() => setShowGenerateConfirm(false)}
        title={intl.formatMessage({
          id: "compliance.portfolio.generate.title",
        })}
        confirmLabel={intl.formatMessage({
          id: "compliance.portfolio.generate.confirm",
        })}
      >
        {intl.formatMessage(
          { id: "compliance.portfolio.generate.description" },
          { count: items.length },
        )}
      </ConfirmationDialog>
    </div>
  );
}
