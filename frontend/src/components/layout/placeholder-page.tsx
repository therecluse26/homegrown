import { PageTitle } from "@/components/common";

export function PlaceholderPage({ title, subtitle }: { title: string; subtitle?: string }) {
  return (
    <div className="py-8">
      <PageTitle title={title} subtitle={subtitle ?? "Coming soon"} />
      <div className="mt-8 bg-surface-container-low rounded-radius-lg p-card-padding">
        <p className="type-body-md text-on-surface-variant">
          This page is under construction.
        </p>
      </div>
    </div>
  );
}
