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
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useStudents } from "@/hooks/use-family";
import {
  useCreatePortfolio,
  type PortfolioOrganization,
} from "@/hooks/use-compliance";

export function PortfolioNew() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [title, setTitle] = useState("");
  const [dateStart, setDateStart] = useState("");
  const [dateEnd, setDateEnd] = useState("");
  const [organization, setOrganization] =
    useState<PortfolioOrganization>("chronological");

  const effectiveStudent =
    studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const createPortfolio = useCreatePortfolio();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !title.trim() || !dateStart || !dateEnd) return;
    createPortfolio.mutate(
      {
        student_id: effectiveStudent,
        title: title.trim(),
        date_range_start: dateStart,
        date_range_end: dateEnd,
        organization,
      },
      {
        onSuccess: (result) => {
          void navigate(
            `/compliance/portfolios/${effectiveStudent}/${result.id}`,
          );
        },
      },
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "portfolioNew.title" })}
      />

      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/compliance/portfolios")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="portfolioNew.title" />
        </h1>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          <div>
            <label
              htmlFor="portfolio-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="portfolio-student"
                value={effectiveStudent}
                onChange={(e) => setStudentId(e.target.value)}
                required
              >
                <option value="">
                  {intl.formatMessage({ id: "activityLog.selectStudent" })}
                </option>
                {students?.map((s) => (
                  <option key={s.id} value={s.id ?? ""}>
                    {s.display_name}
                  </option>
                ))}
              </Select>
            )}
          </div>

          {/* Title */}
          <div>
            <label
              htmlFor="portfolio-title"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="portfolioNew.portfolioTitle" />
            </label>
            <Input
              id="portfolio-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "portfolioNew.portfolioTitle.placeholder",
              })}
              required
            />
          </div>

          {/* Date range */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="date-start"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="portfolioNew.startDate" />
              </label>
              <Input
                id="date-start"
                type="date"
                value={dateStart}
                onChange={(e) => setDateStart(e.target.value)}
                required
              />
            </div>
            <div>
              <label
                htmlFor="date-end"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="portfolioNew.endDate" />
              </label>
              <Input
                id="date-end"
                type="date"
                value={dateEnd}
                onChange={(e) => setDateEnd(e.target.value)}
                required
              />
            </div>
          </div>

          {/* Organization */}
          <div>
            <label
              htmlFor="portfolio-org"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="portfolioNew.organization" />
            </label>
            <Select
              id="portfolio-org"
              value={organization}
              onChange={(e) =>
                setOrganization(e.target.value as PortfolioOrganization)
              }
            >
              <option value="chronological">
                {intl.formatMessage({ id: "portfolioNew.org.chronological" })}
              </option>
              <option value="by_subject">
                {intl.formatMessage({ id: "portfolioNew.org.bySubject" })}
              </option>
              <option value="by_type">
                {intl.formatMessage({ id: "portfolioNew.org.byType" })}
              </option>
            </Select>
          </div>

          {/* Actions */}
          <div className="flex gap-2 justify-end pt-2">
            <Button
              variant="tertiary"
              size="sm"
              type="button"
              onClick={() => void navigate("/compliance/portfolios")}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              size="sm"
              type="submit"
              loading={createPortfolio.isPending}
              disabled={
                !effectiveStudent ||
                !title.trim() ||
                !dateStart ||
                !dateEnd
              }
            >
              <FormattedMessage id="portfolioNew.create" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
