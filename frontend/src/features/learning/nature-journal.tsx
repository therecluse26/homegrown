import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Leaf, ArrowLeft, Sun, Cloud, CloudRain, Wind } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
  Textarea,
} from "@/components/ui";
import { FileUpload } from "@/components/ui/file-upload";
import { SubjectPicker } from "@/components/common/subject-picker";
import { useStudents } from "@/hooks/use-family";
import { useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Types ──────────────────────────────────────────────────────────────────

type ObservationType = "nature_walk" | "backyard" | "garden" | "pond" | "field_trip" | "window";
type WeatherCondition = "sunny" | "partly_cloudy" | "cloudy" | "rainy" | "windy" | "foggy";

const OBSERVATION_TYPES: { value: ObservationType; labelId: string }[] = [
  { value: "nature_walk",  labelId: "methodologyTools.natureJournal.obsType.nature_walk" },
  { value: "backyard",     labelId: "methodologyTools.natureJournal.obsType.backyard" },
  { value: "garden",       labelId: "methodologyTools.natureJournal.obsType.garden" },
  { value: "pond",         labelId: "methodologyTools.natureJournal.obsType.pond" },
  { value: "field_trip",   labelId: "methodologyTools.natureJournal.obsType.field_trip" },
  { value: "window",       labelId: "methodologyTools.natureJournal.obsType.window" },
];

const WEATHER_CONDITIONS: { value: WeatherCondition; icon: typeof Sun; labelId: string }[] = [
  { value: "sunny",         icon: Sun,        labelId: "methodologyTools.natureJournal.weather.sunny" },
  { value: "partly_cloudy", icon: Cloud,      labelId: "methodologyTools.natureJournal.weather.partly_cloudy" },
  { value: "cloudy",        icon: Cloud,      labelId: "methodologyTools.natureJournal.weather.cloudy" },
  { value: "rainy",         icon: CloudRain,  labelId: "methodologyTools.natureJournal.weather.rainy" },
  { value: "windy",         icon: Wind,       labelId: "methodologyTools.natureJournal.weather.windy" },
  { value: "foggy",         icon: Cloud,      labelId: "methodologyTools.natureJournal.weather.foggy" },
];

// ─── Methodology gate banner ─────────────────────────────────────────────────

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "charlotte-mason") return null;
  return (
    <div
      className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6"
      role="note"
    >
      <Icon icon={Leaf} size="sm" className="mt-0.5 shrink-0 text-tertiary" aria-hidden />
      <p className="type-body-sm">
        <FormattedMessage id="methodologyTools.notPrimary.natureJournal" />
      </p>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function NatureJournal() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [observationType, setObservationType] = useState<ObservationType>("nature_walk");
  const [species, setSpecies] = useState("");
  const [weather, setWeather] = useState<WeatherCondition>("sunny");
  const [temperature, setTemperature] = useState("");
  const [location, setLocation] = useState("");
  const [observations, setObservations] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>(["nature_study"]);
  const [durationMinutes, setDurationMinutes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));
  const [photoFiles, setPhotoFiles] = useState<File[]>([]);

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.natureJournal.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !observations.trim()) return;

    const obsTypeLabel = intl.formatMessage({ id: OBSERVATION_TYPES.find(t => t.value === observationType)?.labelId ?? "" });
    const weatherLabel = intl.formatMessage({ id: WEATHER_CONDITIONS.find(w => w.value === weather)?.labelId ?? "" });

    const descriptionParts: string[] = [
      `Type: ${obsTypeLabel}`,
      `Weather: ${weatherLabel}${temperature ? ` · ${temperature}` : ""}`,
      location ? `Location: ${location}` : "",
      species ? `Observed: ${species}` : "",
      observations,
      photoFiles.length > 0 ? `Photos: ${photoFiles.map(f => f.name).join(", ")}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Nature Journal: ${obsTypeLabel}${species ? ` — ${species}` : ""}`,
        description: descriptionParts.join("\n"),
        subject_tags: subjectTags.length > 0 ? subjectTags : ["nature_study"],
        tool_id: "nature-journal",
        duration_minutes: durationMinutes ? Number(durationMinutes) : undefined,
        activity_date: entryDate ? `${entryDate}T00:00:00Z` : undefined,
      },
      { onSuccess: () => { void navigate("/learning/activities"); } },
    );
  }

  if (studentsLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-4" aria-busy="true">
        <Skeleton height="h-8" width="w-48" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate("/learning")}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
        </Button>
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-sm text-on-surface font-semibold flex items-center gap-2"
        >
          <Icon icon={Leaf} size="md" className="text-tertiary" aria-hidden />
          <FormattedMessage id="methodologyTools.natureJournal.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          {students && students.length > 1 && (
            <div>
              <label htmlFor="nj-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select
                id="nj-student"
                value={studentId}
                onChange={e => setStudentId(e.target.value)}
                required
              >
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => (
                  <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>
                ))}
              </Select>
            </div>
          )}

          {/* Observation type */}
          <div>
            <label htmlFor="nj-type" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.natureJournal.observationType" />
            </label>
            <Select
              id="nj-type"
              value={observationType}
              onChange={e => setObservationType(e.target.value as ObservationType)}
            >
              {OBSERVATION_TYPES.map(t => (
                <option key={t.value} value={t.value}>{intl.formatMessage({ id: t.labelId })}</option>
              ))}
            </Select>
          </div>

          {/* Date + Duration */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="nj-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="nj-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="nj-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="nj-duration"
                type="number"
                min="1"
                max="480"
                placeholder="30"
                value={durationMinutes}
                onChange={e => setDurationMinutes(e.target.value)}
              />
            </div>
          </div>

          {/* Weather */}
          <fieldset>
            <legend className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.natureJournal.weather" />
            </legend>
            <div className="flex flex-wrap gap-2" role="radiogroup">
              {WEATHER_CONDITIONS.map(w => (
                <label
                  key={w.value}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full cursor-pointer type-label-md transition-colors ${
                    weather === w.value
                      ? "bg-primary text-on-primary"
                      : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                  }`}
                >
                  <input
                    type="radio"
                    name="nj-weather"
                    value={w.value}
                    checked={weather === w.value}
                    onChange={() => setWeather(w.value)}
                    className="sr-only"
                  />
                  <Icon icon={w.icon} size="xs" aria-hidden />
                  {intl.formatMessage({ id: w.labelId })}
                </label>
              ))}
            </div>
          </fieldset>

          {/* Temperature + Location */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="nj-temp" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.natureJournal.temperature" />
              </label>
              <Input id="nj-temp" placeholder="72°F" value={temperature} onChange={e => setTemperature(e.target.value)} />
            </div>
            <div>
              <label htmlFor="nj-location" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.natureJournal.location" />
              </label>
              <Input
                id="nj-location"
                placeholder={intl.formatMessage({ id: "methodologyTools.natureJournal.locationPlaceholder" })}
                value={location}
                onChange={e => setLocation(e.target.value)}
              />
            </div>
          </div>

          {/* Species */}
          <div>
            <label htmlFor="nj-species" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.natureJournal.species" />
            </label>
            <Input
              id="nj-species"
              placeholder={intl.formatMessage({ id: "methodologyTools.natureJournal.speciesPlaceholder" })}
              value={species}
              onChange={e => setSpecies(e.target.value)}
            />
          </div>

          {/* Observation narrative */}
          <div>
            <label htmlFor="nj-observations" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.natureJournal.observations" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Textarea
              id="nj-observations"
              placeholder={intl.formatMessage({ id: "methodologyTools.natureJournal.observationsPlaceholder" })}
              value={observations}
              onChange={e => setObservations(e.target.value)}
              rows={5}
              required
            />
          </div>

          {/* Drawing / photo upload */}
          <div>
            <p className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.natureJournal.drawingPhoto" />
            </p>
            <FileUpload
              accept="image/*"
              multiple
              onFiles={files => setPhotoFiles(prev => [...prev, ...files])}
            />
            {photoFiles.length > 0 && (
              <ul className="flex flex-wrap gap-2 mt-2">
                {photoFiles.map((f, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-1 px-3 py-1 rounded-full bg-secondary-container text-on-secondary-container type-label-sm"
                  >
                    {f.name}
                    <button
                      type="button"
                      onClick={() => setPhotoFiles(prev => prev.filter((_, idx) => idx !== i))}
                      className="ml-1 hover:text-error transition-colors"
                      aria-label={`Remove ${f.name}`}
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Subject tags */}
          <div>
            <p className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.field.subjectTags" />
            </p>
            <SubjectPicker value={subjectTags} onChange={setSubjectTags} />
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button variant="tertiary" type="button" onClick={() => void navigate("/learning")}>
              <FormattedMessage id="action.cancel" />
            </Button>
            <Button
              variant="primary"
              type="submit"
              disabled={!effectiveStudent || !observations.trim() || logActivity.isPending}
              loading={logActivity.isPending}
            >
              <FormattedMessage id="methodologyTools.action.saveEntry" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
