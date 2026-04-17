import { useEffect, useRef, useState } from "react";
import { useParams, Link } from "react-router";
import { Share2, RotateCcw, Trophy } from "lucide-react";
import { Badge, Button, Card, EmptyState, Skeleton } from "@/components/ui";
import { useQuizResult } from "@/hooks/use-discover";

const RANK_COLORS: Record<number, string> = {
  1: "bg-warning-container text-on-warning-container",
  2: "bg-surface-container-high text-on-surface",
  3: "bg-secondary-container text-on-secondary-container",
};

export function QuizResults() {
  const { shareId } = useParams<{ shareId: string }>();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: result, isPending, error } = useQuizResult(shareId);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    document.title = "Your Quiz Results - Homegrown Academy";
    headingRef.current?.focus();
  }, []);

  function handleShare() {
    void navigator.clipboard.writeText(window.location.href).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  if (isPending) {
    return (
      <div className="space-y-6">
        <Skeleton width="w-64" height="h-8" />
        <Skeleton width="w-full" height="h-24" />
        <Skeleton width="w-full" height="h-24" />
        <Skeleton width="w-full" height="h-24" />
      </div>
    );
  }

  if (error || !result?.recommendations?.length) {
    return (
      <EmptyState
        message="Results not found"
        description="This quiz result may have expired or the link may be incorrect."
        action={
          <Link to="/discover/quiz" tabIndex={-1}>
            <Button>Take the Quiz</Button>
          </Link>
        }
      />
    );
  }

  const recommendations = result.recommendations;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center space-y-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          Your Homeschooling Match
        </h1>
        <p className="type-body-md text-on-surface-variant">
          Based on your answers, here are the methodologies that best fit your
          family.
        </p>
      </div>

      {/* Ranked results */}
      <div className="space-y-4">
        {recommendations.map((rec) => {
          const rank = rec.rank ?? 0;
          const score = rec.score_percentage ?? 0;
          const rankColor = RANK_COLORS[rank] ?? "bg-surface-container text-on-surface";

          return (
            <Card key={rec.methodology_slug} className="space-y-3">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-3">
                  <span
                    className={`inline-flex h-8 w-8 items-center justify-center rounded-full type-label-lg font-bold ${rankColor}`}
                  >
                    {rank}
                  </span>
                  <div>
                    <h2 className="type-title-md text-on-surface font-semibold">
                      {rec.methodology_name}
                    </h2>
                  </div>
                </div>
                <Badge variant={rank === 1 ? "success" : "default"}>
                  {score}% match
                </Badge>
              </div>

              {/* Score bar */}
              <div className="h-2 w-full overflow-hidden rounded-full bg-tertiary-fixed">
                <div
                  className="h-full rounded-full bg-primary transition-all duration-500"
                  style={{ width: `${String(score)}%` }}
                />
              </div>

              {rec.explanation && (
                <p className="type-body-sm text-on-surface-variant">
                  {rec.explanation}
                </p>
              )}
            </Card>
          );
        })}
      </div>

      {/* Actions */}
      <div className="flex flex-col items-center gap-4 pt-2">
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={handleShare}
            leadingIcon={<Share2 className="h-4 w-4" />}
          >
            {copied ? "Link Copied!" : "Share Results"}
          </Button>
          <Link to="/discover/quiz" tabIndex={-1}>
            <Button
              variant="tertiary"
              leadingIcon={<RotateCcw className="h-4 w-4" />}
            >
              Retake Quiz
            </Button>
          </Link>
        </div>

        <div className="text-center space-y-2 pt-4">
          <div className="flex items-center justify-center gap-2 text-primary">
            <Trophy className="h-5 w-5" />
            <p className="type-title-sm font-semibold">
              Ready to start your journey?
            </p>
          </div>
          <Link to="/auth/register" tabIndex={-1}>
            <Button variant="gradient" size="lg">
              Create Your Free Account
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
