import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink, useNavigate } from "react-router";
import { Plus, FolderOpen, Download, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
  Badge,
  Input,
  EmptyState,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import { useStudents } from "@/hooks/use-family";
import {
  usePortfolios,
  useCreatePortfolio,
  useDeletePortfolio,
} from "@/hooks/use-compliance";
import type {
  PortfolioSummary,
  PortfolioOrganization,
} from "@/hooks/use-compliance";

// ─── Status badge ──────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: PortfolioSummary["status"] }) {
  const variant =
    status === "ready" ? "primary" : status === "generating" ? "secondary" : undefined;
  return (
    <Badge variant={variant}>
      <FormattedMessage id={`compliance.portfolio.status.${status}`} />
    </Badge>
  );
}

// ─── Portfolio card ────────────────────────────────────────────────────────

function PortfolioCard({
  portfolio,
  onDelete,
}: {
  portfolio: PortfolioSummary;
  onDelete: (id: string) => void;
}) {
  const intl = useIntl();
  const startDate = new Date(portfolio.date_range_start + "T12:00:00");
  const endDate = new Date(portfolio.date_range_end + "T12:00:00");
  const dateRange = `${startDate.toLocaleDateString(intl.locale, {
    month: "short",
    day: "numeric",
  })} – ${endDate.toLocaleDateString(intl.locale, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })}`;

  return (
    <Card className="p-card-padding">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <RouterLink
              to={`/compliance/portfolios/${portfolio.id}`}
              className="type-title-sm text-on-surface font-semibold hover:text-primary transition-colors"
            >
              {portfolio.title}
            </RouterLink>
            <StatusBadge status={portfolio.status} />
          </div>
          <p className="type-body-sm text-on-surface-variant">
            {portfolio.student_name} — {dateRange}
          </p>
          <p className="type-label-sm text-on-surface-variant mt-1">
            <FormattedMessage
              id="compliance.portfolio.itemCount"
              values={{ count: portfolio.item_count }}
            />
          </p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {portfolio.download_url && (
            <a
              href={portfolio.download_url}
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors touch-target"
              aria-label={intl.formatMessage({
                id: "compliance.portfolio.download",
              })}
            >
              <Icon icon={Download} size="sm" />
            </a>
          )}
          <button
            onClick={() => onDelete(portfolio.id)}
            className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-error-container hover:text-on-error-container transition-colors touch-target"
            aria-label={intl.formatMessage(
              { id: "compliance.portfolio.delete.label" },
              { name: portfolio.title },
            )}
          >
            <Icon icon={Trash2} size="sm" />
          </button>
        </div>
      </div>
    </Card>
  );
}

// ─── Create form ───────────────────────────────────────────────────────────

function CreatePortfolioForm({
  students,
  onClose,
}: {
  students: { id: string; display_name: string }[];
  onClose: () => void;
}) {
  const intl = useIntl();
  const navigate = useNavigate();
  const createPortfolio = useCreatePortfolio();

  const [title, setTitle] = useState("");
  const [studentId, setStudentId] = useState(students[0]?.id ?? "");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [organization, setOrganization] =
    useState<PortfolioOrganization>("chronological");

  const canSubmit = title.trim() && studentId && startDate && endDate;

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!canSubmit) return;
      createPortfolio.mutate(
        {
          student_id: studentId,
          title: title.trim(),
          date_range_start: startDate,
          date_range_end: endDate,
          organization,
        },
        {
          onSuccess: (data) => {
            navigate(`/compliance/portfolios/${data.id}`);
          },
        },
      );
    },
    [canSubmit, studentId, title, startDate, endDate, organization, createPortfolio, navigate],
  );

  return (
    <Card className="p-card-padding mb-6">
      <h2 className="type-title-sm text-on-surface mb-4">
        <FormattedMessage id="compliance.portfolio.create.title" />
      </h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="portfolio-title"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.form.title" />
            </label>
            <Input
              id="portfolio-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "compliance.portfolio.form.title.placeholder",
              })}
            />
          </div>
          <div>
            <label
              htmlFor="portfolio-student"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.form.student" />
            </label>
            <Select
              id="portfolio-student"
              value={studentId}
              onChange={(e) => setStudentId(e.target.value)}
            >
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.display_name}
                </option>
              ))}
            </Select>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label
              htmlFor="portfolio-start"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="planning.export.startDate" />
            </label>
            <input
              id="portfolio-start"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          </div>
          <div>
            <label
              htmlFor="portfolio-end"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="planning.export.endDate" />
            </label>
            <input
              id="portfolio-end"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          </div>
          <div>
            <label
              htmlFor="portfolio-org"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.form.organization" />
            </label>
            <Select
              id="portfolio-org"
              value={organization}
              onChange={(e) =>
                setOrganization(e.target.value as PortfolioOrganization)
              }
            >
              <option value="chronological">
                {intl.formatMessage({ id: "compliance.portfolio.org.chronological" })}
              </option>
              <option value="by_subject">
                {intl.formatMessage({ id: "compliance.portfolio.org.bySubject" })}
              </option>
              <option value="by_type">
                {intl.formatMessage({ id: "compliance.portfolio.org.byType" })}
              </option>
            </Select>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="tertiary" size="sm" onClick={onClose}>
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={!canSubmit || createPortfolio.isPending}
          >
            <FormattedMessage id="compliance.portfolio.create.submit" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function PortfolioList() {
  const intl = useIntl();
  const { tier } = useAuth();
  const [showCreate, setShowCreate] = useState(false);
  const [studentFilter, setStudentFilter] = useState<string | undefined>();
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const { data: students } = useStudents();
  const { data: portfolios, isPending } = usePortfolios(studentFilter);
  const deletePortfolio = useDeletePortfolio();

  const handleDelete = useCallback(() => {
    if (!deleteTarget) return;
    deletePortfolio.mutate(deleteTarget, {
      onSuccess: () => setDeleteTarget(null),
    });
  }, [deleteTarget, deletePortfolio]);

  if (tier === "free") {
    return <TierGate featureName="Portfolio Builder" />;
  }

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "compliance.portfolio.pageTitle" })}
      />

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-3">
          {students && students.length > 1 && (
            <Select
              value={studentFilter ?? ""}
              onChange={(e) =>
                setStudentFilter(e.target.value || undefined)
              }
              className="w-40"
              aria-label={intl.formatMessage({
                id: "compliance.portfolio.studentFilter",
              })}
            >
              <option value="">
                {intl.formatMessage({ id: "planning.export.allStudents" })}
              </option>
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.display_name}
                </option>
              ))}
            </Select>
          )}
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowCreate(true)}
        >
          <Icon icon={Plus} size="sm" className="mr-1" />
          <FormattedMessage id="compliance.portfolio.create" />
        </Button>
      </div>

      {/* Create form */}
      {showCreate && students && students.length > 0 && (
        <CreatePortfolioForm
          students={students
            .filter((s): s is typeof s & { id: string; display_name: string } =>
              !!s.id && !!s.display_name
            )}
          onClose={() => setShowCreate(false)}
        />
      )}

      {/* Portfolio list */}
      {isPending ? (
        <div className="space-y-3">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
          ))}
        </div>
      ) : !portfolios || portfolios.length === 0 ? (
        <EmptyState
          illustration={<Icon icon={FolderOpen} size="xl" />}
          message={intl.formatMessage({
            id: "compliance.portfolio.empty",
          })}
          description={intl.formatMessage({
            id: "compliance.portfolio.empty.description",
          })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowCreate(true)}
            >
              <Icon icon={Plus} size="sm" className="mr-1" />
              <FormattedMessage id="compliance.portfolio.create" />
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {portfolios.map((p) => (
            <PortfolioCard
              key={p.id}
              portfolio={p}
              onDelete={setDeleteTarget}
            />
          ))}
        </div>
      )}

      {/* Delete confirmation */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onConfirm={handleDelete}
        onClose={() => setDeleteTarget(null)}
        title={intl.formatMessage({ id: "compliance.portfolio.delete.title" })}
        confirmLabel={intl.formatMessage({
          id: "compliance.portfolio.delete.confirm",
        })}
        destructive
      >
        {intl.formatMessage({
          id: "compliance.portfolio.delete.description",
        })}
      </ConfirmationDialog>
    </div>
  );
}
