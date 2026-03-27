import { useState, useMemo, useCallback, type ReactNode } from "react";
import { ChevronUp, ChevronDown, ChevronsUpDown } from "lucide-react";
import { Icon } from "./icon";

type SortDirection = "asc" | "desc" | null;

type Column<T> = {
  id: string;
  header: string;
  /** Render cell content. Receives the row data. */
  cell: (row: T) => ReactNode;
  /** Sort accessor — return a string or number for comparison */
  sortValue?: (row: T) => string | number;
  /** Column width class (optional) */
  width?: string;
};

type DataTableProps<T> = {
  columns: Column<T>[];
  data: T[];
  /** Unique key accessor for each row */
  getRowId: (row: T) => string;
  /** Optional empty state message */
  emptyMessage?: string;
  className?: string;
};

export function DataTable<T>({
  columns,
  data,
  getRowId,
  emptyMessage = "No data available",
  className = "",
}: DataTableProps<T>) {
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);

  const handleSort = useCallback(
    (columnId: string) => {
      if (sortColumn === columnId) {
        // Cycle: asc → desc → null
        if (sortDirection === "asc") setSortDirection("desc");
        else if (sortDirection === "desc") {
          setSortColumn(null);
          setSortDirection(null);
        }
      } else {
        setSortColumn(columnId);
        setSortDirection("asc");
      }
    },
    [sortColumn, sortDirection],
  );

  const sortedData = useMemo(() => {
    if (!sortColumn || !sortDirection) return data;

    const col = columns.find((c) => c.id === sortColumn);
    if (!col?.sortValue) return data;

    const accessor = col.sortValue;
    return [...data].sort((a, b) => {
      const aVal = accessor(a);
      const bVal = accessor(b);
      const cmp = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      return sortDirection === "asc" ? cmp : -cmp;
    });
  }, [data, columns, sortColumn, sortDirection]);

  function getSortIcon(columnId: string) {
    if (sortColumn !== columnId) return ChevronsUpDown;
    return sortDirection === "asc" ? ChevronUp : ChevronDown;
  }

  return (
    <div className={`w-full overflow-x-auto ${className}`}>
      <table className="w-full type-body-md text-on-surface">
        <thead>
          <tr className="bg-surface-container-low">
            {columns.map((col) => (
              <th
                key={col.id}
                className={`px-4 py-3 text-left type-label-lg text-on-surface-variant font-medium ${col.width ?? ""}`}
              >
                {col.sortValue ? (
                  <button
                    className="inline-flex items-center gap-1 hover:text-on-surface transition-colors"
                    onClick={() => handleSort(col.id)}
                    aria-sort={
                      sortColumn === col.id && sortDirection
                        ? sortDirection === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    {col.header}
                    <Icon icon={getSortIcon(col.id)} size="xs" />
                  </button>
                ) : (
                  col.header
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.length === 0 ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-4 py-12 text-center type-body-md text-on-surface-variant"
              >
                {emptyMessage}
              </td>
            </tr>
          ) : (
            sortedData.map((row) => (
              <tr
                key={getRowId(row)}
                className="transition-colors hover:bg-surface-container-low"
              >
                {columns.map((col) => (
                  <td key={col.id} className={`px-4 py-3 ${col.width ?? ""}`}>
                    {col.cell(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
