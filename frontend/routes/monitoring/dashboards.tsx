import { define } from "@/utils.ts";
import MonitoringDashboards from "@/islands/MonitoringDashboards.tsx";

export default define.page(function DashboardsPage() {
  return (
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">
          Dashboards
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Provisioned Grafana dashboards
        </p>
      </div>
      <MonitoringDashboards />
    </div>
  );
});
