import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router";
import { Search, MapPin } from "lucide-react";
import { Badge, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { useStateGuides } from "@/hooks/use-discover";

export function StateGuides() {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: guides, isPending, error } = useStateGuides();
  const [search, setSearch] = useState("");

  useEffect(() => {
    document.title = "State Homeschooling Guides - Homegrown Academy";
    headingRef.current?.focus();
  }, []);

  const filtered = useMemo(() => {
    if (!guides) return [];
    const term = search.toLowerCase().trim();
    if (!term) return guides;
    return guides.filter(
      (g) =>
        g.state_name?.toLowerCase().includes(term) ||
        g.state_code?.toLowerCase().includes(term),
    );
  }, [guides, search]);

  if (isPending) {
    return (
      <div className="space-y-6">
        <Skeleton width="w-64" height="h-8" />
        <Skeleton width="w-full" height="h-10" />
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 12 }, (_, i) => (
            <Skeleton key={i} width="w-full" height="h-16" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <EmptyState
        message="Unable to load state guides"
        description="Please try again later."
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          State Homeschooling Guides
        </h1>
        <p className="type-body-md text-on-surface-variant">
          Find the legal requirements and resources for homeschooling in your
          state.
        </p>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-on-surface-variant" />
        <Input
          type="text"
          placeholder="Search by state name..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-10"
          aria-label="Filter states"
        />
      </div>

      {/* State grid */}
      {filtered.length === 0 ? (
        <EmptyState
          message="No states match your search"
          description="Try a different search term."
          illustration={<MapPin className="h-12 w-12" />}
        />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((state) => {
            const isAvailable = state.is_available;
            const content = (
              <Card
                interactive={isAvailable}
                className={`flex items-center justify-between ${
                  !isAvailable ? "opacity-60" : ""
                }`}
              >
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4 text-on-surface-variant" />
                  <span className="type-body-md text-on-surface font-medium">
                    {state.state_name}
                  </span>
                </div>
                {isAvailable ? (
                  <Badge variant="success">Available</Badge>
                ) : (
                  <Badge variant="default">Coming Soon</Badge>
                )}
              </Card>
            );

            if (isAvailable) {
              return (
                <Link
                  key={state.state_code}
                  to={`/discover/states/${state.state_code ?? ""}`}
                  className="block focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  {content}
                </Link>
              );
            }

            return (
              <div key={state.state_code} aria-disabled="true">
                {content}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
