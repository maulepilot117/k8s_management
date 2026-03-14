import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { IS_BROWSER } from "fresh/runtime";
import { apiGet } from "@/lib/api.ts";

interface GrafanaDashboard {
  uid: string;
  title: string;
  url: string;
  tags: string[];
  type: string;
}

export default function MonitoringDashboards() {
  const dashboards = useSignal<GrafanaDashboard[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);

  useEffect(() => {
    if (!IS_BROWSER) return;

    apiGet<GrafanaDashboard[]>("/v1/monitoring/dashboards")
      .then((res) => {
        dashboards.value = res.data;
      })
      .catch((err) => {
        error.value = err.message ?? "Failed to load dashboards";
      })
      .finally(() => {
        loading.value = false;
      });
  }, []);

  if (!IS_BROWSER) return null;

  if (loading.value) {
    return (
      <div class="flex items-center justify-center p-12">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-brand" />
      </div>
    );
  }

  if (error.value) {
    return (
      <div class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
        {error.value}
      </div>
    );
  }

  if (dashboards.value.length === 0) {
    return (
      <div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">
        <p class="text-lg font-medium text-slate-500 dark:text-slate-400">
          No Dashboards Found
        </p>
        <p class="mt-1">
          Dashboards are provisioned when Grafana is detected and configured.
        </p>
        <a
          href="/monitoring"
          class="mt-3 inline-block text-brand hover:underline"
        >
          Check monitoring status
        </a>
      </div>
    );
  }

  return (
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {dashboards.value.map((d) => (
        <a
          key={d.uid}
          href={`/api/v1/monitoring/grafana/proxy${d.url}?kiosk=1`}
          target="_blank"
          rel="noopener noreferrer"
          class="group rounded-lg border border-slate-200 bg-white p-4 transition-colors hover:border-brand dark:border-slate-700 dark:bg-slate-800 dark:hover:border-brand"
        >
          <h3 class="font-medium text-slate-900 group-hover:text-brand dark:text-white">
            {d.title}
          </h3>
          {d.tags && d.tags.length > 0 && (
            <div class="mt-2 flex flex-wrap gap-1">
              {d.tags.map((tag) => (
                <span
                  key={tag}
                  class="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-400"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}
          <p class="mt-2 text-xs text-slate-400 dark:text-slate-500">
            UID: {d.uid}
          </p>
        </a>
      ))}
    </div>
  );
}
