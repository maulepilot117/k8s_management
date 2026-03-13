import { useSignal } from "@preact/signals";
import { IS_BROWSER } from "fresh/runtime";
import { apiGet } from "@/lib/api.ts";

interface QueryResult {
  resultType: string;
  result: Array<{
    metric: Record<string, string>;
    value?: [number, string];
    values?: Array<[number, string]>;
  }>;
  warnings?: string[];
}

const TIME_RANGES = [
  { label: "Last 1h", value: "1h" },
  { label: "Last 6h", value: "6h" },
  { label: "Last 24h", value: "24h" },
  { label: "Last 7d", value: "7d" },
];

function subtractDuration(d: string): Date {
  const now = new Date();
  const val = parseInt(d);
  const unit = d.replace(/\d+/, "");
  switch (unit) {
    case "h":
      now.setHours(now.getHours() - val);
      break;
    case "d":
      now.setDate(now.getDate() - val);
      break;
  }
  return now;
}

export default function PromQLQuery() {
  const query = useSignal("");
  const queryType = useSignal<"instant" | "range">("instant");
  const timeRange = useSignal("1h");
  const result = useSignal<QueryResult | null>(null);
  const loading = useSignal(false);
  const error = useSignal<string | null>(null);

  if (!IS_BROWSER) return null;

  async function runQuery() {
    const q = query.value.trim();
    if (!q) return;

    loading.value = true;
    error.value = null;
    result.value = null;

    try {
      if (queryType.value === "instant") {
        const res = await apiGet<QueryResult>(
          `/v1/monitoring/query?query=${encodeURIComponent(q)}`,
        );
        result.value = res.data;
      } else {
        const end = new Date();
        const start = subtractDuration(timeRange.value);
        // Step: ~200 data points
        const stepMs = (end.getTime() - start.getTime()) / 200;
        const step = `${Math.max(Math.round(stepMs / 1000), 15)}s`;

        const params = new URLSearchParams({
          query: q,
          start: start.toISOString(),
          end: end.toISOString(),
          step,
        });
        const res = await apiGet<QueryResult>(
          `/v1/monitoring/query_range?${params}`,
        );
        result.value = res.data;
      }
    } catch (err) {
      error.value = (err as Error).message ?? "Query failed";
    } finally {
      loading.value = false;
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      runQuery();
    }
  }

  return (
    <div class="space-y-4">
      {/* Query input */}
      <div class="space-y-3">
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            PromQL Expression
          </label>
          <textarea
            value={query.value}
            onInput={(e) =>
              query.value = (e.target as HTMLTextAreaElement).value}
            onKeyDown={handleKeyDown}
            placeholder='up{job="prometheus"}'
            rows={3}
            class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand dark:border-slate-600 dark:bg-slate-800 dark:text-white"
          />
          <p class="mt-1 text-xs text-slate-400">
            Press Ctrl+Enter to run query
          </p>
        </div>

        {/* Query type and controls */}
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex rounded-md border border-slate-300 dark:border-slate-600">
            <button
              type="button"
              onClick={() => queryType.value = "instant"}
              class={`px-3 py-1.5 text-sm ${
                queryType.value === "instant"
                  ? "bg-brand text-white"
                  : "text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700"
              }`}
            >
              Instant
            </button>
            <button
              type="button"
              onClick={() => queryType.value = "range"}
              class={`px-3 py-1.5 text-sm border-l border-slate-300 dark:border-slate-600 ${
                queryType.value === "range"
                  ? "bg-brand text-white"
                  : "text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700"
              }`}
            >
              Range
            </button>
          </div>

          {queryType.value === "range" && (
            <select
              value={timeRange.value}
              onChange={(e) =>
                timeRange.value = (e.target as HTMLSelectElement).value}
              class="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300"
            >
              {TIME_RANGES.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          )}

          <button
            type="button"
            onClick={runQuery}
            disabled={loading.value || !query.value.trim()}
            class="rounded-md bg-brand px-4 py-1.5 text-sm font-medium text-white hover:bg-brand/90 disabled:opacity-50"
          >
            {loading.value ? "Running..." : "Run Query"}
          </button>
        </div>
      </div>

      {/* Error */}
      {error.value && (
        <div class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error.value}
        </div>
      )}

      {/* Results */}
      {result.value && (
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
            <span>Type: {result.value.resultType}</span>
            <span>|</span>
            <span>{result.value.result.length} result(s)</span>
            {result.value.warnings && result.value.warnings.length > 0 && (
              <span class="text-amber-500">
                | {result.value.warnings.length} warning(s)
              </span>
            )}
          </div>

          {result.value.warnings?.map((w, i) => (
            <div
              key={i}
              class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
            >
              {w}
            </div>
          ))}

          {result.value.result.length === 0
            ? (
              <p class="py-8 text-center text-sm text-slate-400">
                No results
              </p>
            )
            : (
              <div class="overflow-x-auto">
                <table class="w-full text-sm">
                  <thead>
                    <tr class="border-b border-slate-200 dark:border-slate-700">
                      <th class="px-3 py-2 text-left font-medium text-slate-500 dark:text-slate-400">
                        Metric
                      </th>
                      <th class="px-3 py-2 text-left font-medium text-slate-500 dark:text-slate-400">
                        Value
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.value.result.map((r, i) => {
                      const metricLabel = Object.entries(r.metric)
                        .map(([k, v]) => `${k}="${v}"`)
                        .join(", ");
                      const value = r.value
                        ? r.value[1]
                        : r.values
                        ? `${r.values.length} samples`
                        : "N/A";
                      return (
                        <tr
                          key={i}
                          class="border-b border-slate-100 dark:border-slate-800"
                        >
                          <td class="px-3 py-2 font-mono text-xs text-slate-700 dark:text-slate-300">
                            {`{${metricLabel}}` || "{}"}
                          </td>
                          <td class="px-3 py-2 font-mono text-xs text-slate-900 dark:text-white">
                            {value}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
        </div>
      )}
    </div>
  );
}
