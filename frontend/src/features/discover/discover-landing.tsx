import { useEffect, useRef } from "react";
import { Link } from "react-router";
import { Compass, MapPin, ArrowRight } from "lucide-react";
import { Button, Card } from "@/components/ui";

export function DiscoverLanding() {
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    document.title = "Discover Your Homeschooling Path - Homegrown Academy";
    headingRef.current?.focus();
  }, []);

  return (
    <div className="space-y-8">
      {/* Hero */}
      <div className="text-center space-y-3">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          Discover Your Homeschooling Path
        </h1>
        <p className="type-body-lg text-on-surface-variant max-w-xl mx-auto">
          Whether you&rsquo;re just starting out or looking to refine your
          approach, we&rsquo;ll help you find the methodology and legal
          requirements that fit your family.
        </p>
      </div>

      {/* Feature cards */}
      <div className="grid gap-6 sm:grid-cols-2">
        <Card className="space-y-4">
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary-container p-2.5">
              <Compass className="h-6 w-6 text-on-primary-container" />
            </div>
            <h2 className="type-title-md text-on-surface font-semibold">
              Find Your Methodology
            </h2>
          </div>
          <p className="type-body-md text-on-surface-variant">
            Take our short quiz to discover which homeschooling approach best
            matches your family&rsquo;s values, learning style, and goals.
          </p>
          <Link to="/discover/quiz" tabIndex={-1}>
            <Button
              variant="primary"
              trailingIcon={<ArrowRight className="h-4 w-4" />}
            >
              Take the Quiz
            </Button>
          </Link>
        </Card>

        <Card className="space-y-4">
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-secondary-container p-2.5">
              <MapPin className="h-6 w-6 text-on-secondary-container" />
            </div>
            <h2 className="type-title-md text-on-surface font-semibold">
              State Legal Guides
            </h2>
          </div>
          <p className="type-body-md text-on-surface-variant">
            Understand the homeschooling laws and requirements in your state,
            from notification to record-keeping and assessments.
          </p>
          <Link to="/discover/states" tabIndex={-1}>
            <Button
              variant="secondary"
              trailingIcon={<ArrowRight className="h-4 w-4" />}
            >
              Browse State Guides
            </Button>
          </Link>
        </Card>
      </div>

      {/* Bottom CTA */}
      <div className="text-center space-y-3 pt-4">
        <p className="type-body-md text-on-surface-variant">
          Ready to start your homeschooling journey?
        </p>
        <Link to="/auth/register" tabIndex={-1}>
          <Button variant="gradient" size="lg">
            Create Your Free Account
          </Button>
        </Link>
      </div>
    </div>
  );
}
