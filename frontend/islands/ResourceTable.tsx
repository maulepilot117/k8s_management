import { useComputed, useSignal } from "@preact/signals";
import { useCallback, useEffect } from "preact/hooks";
import { IS_BROWSER } from "fresh/runtime";
import { apiGet } from "@/lib/api.ts";
import { selectedNamespace } from "@/lib/namespace.ts";
import { subscribe } from "@/lib/ws.ts";
import { RESOURCE_COLUMNS } from "@/lib/resource-columns.ts";
import { DataTable } from "@/components/ui/DataTable.tsx";
import { SearchBar } from "@/components/ui/SearchBar.tsx";
import type { K8sResource } from "@/lib/k8s-types.ts";

interface ResourceTableProps {
  /** API kind string matching backend route, e.g. "pods", "deployments" */
  kind: string;
  /** Display title for the page header */
  title: string;
  /** Whether this resource is cluster-scoped (no namespace filtering) */
  clusterScoped?: boolean;
  /** Whether to subscribe to WebSocket events (false for secrets) */
  enableWS?: boolean;
}

export default function ResourceTable({
  kind,
  title,
  clusterScoped = false,
  enableWS = true,
}: ResourceTableProps) {
  const items = useSignal<K8sResource[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const search = useSignal("");
  const sortKey = useSignal("name");
  const sortDir = useSignal<"asc" | "desc">("asc");

  const columns = RESOURCE_COLUMNS[kind] ?? [];

  // Namespace for API calls
  const ns = useComputed(() =>
    clusterScoped
      ? ""
      : (selectedNamespace.value === "all" ? "" : selectedNamespace.value)
  );

  // Fetch resources via REST
  const fetchResources = useCallback(async () => {
    loading.value = true;
    error.value = null;
    try {
      const path = ns.value
        ? `/v1/resources/${kind}/${ns.value}`
        : `/v1/resources/${kind}`;
      const res = await apiGet<K8sResource[]>(path);
      items.value = Array.isArray(res.data) ? res.data : [];
    } catch (err) {
      error.value = err instanceof Error
        ? err.message
        : "Failed to load resources";
      items.value = [];
    } finally {
      loading.value = false;
    }
  }, [kind]);

  // Re-fetch when namespace changes
  useEffect(() => {
    if (!IS_BROWSER) return;
    fetchResources();
  }, [ns.value]);

  // Subscribe to WebSocket events for real-time updates
  useEffect(() => {
    if (!IS_BROWSER || !enableWS) return;

    const subId = `${kind}-${ns.value || "all"}`;
    const unsubscribe = subscribe(
      subId,
      kind,
      ns.value,
      (eventType, object) => {
        if (!object || typeof object !== "object") return;
        const resource = object as K8sResource;
        const uid = resource.metadata?.uid;
        if (!uid) return;

        switch (eventType) {
          case "ADDED": {
            // Only add if not already present
            if (!items.value.some((r) => r.metadata.uid === uid)) {
              items.value = [...items.value, resource];
            }
            break;
          }
          case "MODIFIED": {
            items.value = items.value.map((r) =>
              r.metadata.uid === uid ? resource : r
            );
            break;
          }
          case "DELETED": {
            items.value = items.value.filter((r) => r.metadata.uid !== uid);
            break;
          }
          case "RBAC_DENIED": {
            // Silently ignore — the REST fetch already handles permissions
            break;
          }
        }
      },
    );

    return unsubscribe;
  }, [kind, ns.value, enableWS]);

  // Client-side filter + sort
  const displayed = useComputed(() => {
    let filtered = items.value;

    // Search filter
    const q = search.value.toLowerCase().trim();
    if (q) {
      filtered = filtered.filter((r) => {
        const name = r.metadata.name.toLowerCase();
        const namespace = (r.metadata.namespace ?? "").toLowerCase();
        return name.includes(q) || namespace.includes(q);
      });
    }

    // Sort
    const key = sortKey.value;
    const dir = sortDir.value === "asc" ? 1 : -1;
    return [...filtered].sort((a, b) => {
      let va: string;
      let vb: string;
      if (key === "name") {
        va = a.metadata.name;
        vb = b.metadata.name;
      } else if (key === "namespace") {
        va = a.metadata.namespace ?? "";
        vb = b.metadata.namespace ?? "";
      } else if (key === "age") {
        // Sort by creation timestamp (newer first if desc)
        va = a.metadata.creationTimestamp;
        vb = b.metadata.creationTimestamp;
      } else {
        va = a.metadata.name;
        vb = b.metadata.name;
      }
      return va < vb ? -dir : va > vb ? dir : 0;
    });
  });

  const handleSort = (key: string) => {
    if (sortKey.value === key) {
      sortDir.value = sortDir.value === "asc" ? "desc" : "asc";
    } else {
      sortKey.value = key;
      sortDir.value = "asc";
    }
  };

  return (
    <div class="space-y-4">
      {/* Header */}
      <div class="flex items-center justify-between">
        <h1 class="text-xl font-semibold text-slate-900 dark:text-white">
          {title}
        </h1>
        <div class="flex items-center gap-3">
          <span class="text-sm text-slate-500 dark:text-slate-400">
            {loading.value ? "Loading..." : `${displayed.value.length} items`}
          </span>
          <button
            type="button"
            onClick={fetchResources}
            class="rounded-md p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-700 dark:hover:text-slate-300"
            title="Refresh"
          >
            <svg
              class={`h-4 w-4 ${loading.value ? "animate-spin" : ""}`}
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
            >
              <path d="M14 8A6 6 0 1 1 8 2" />
              <path d="M14 2v4h-4" />
            </svg>
          </button>
        </div>
      </div>

      {/* Search */}
      <div class="max-w-sm">
        <SearchBar
          value={search.value}
          onInput={(v) => {
            search.value = v;
          }}
          placeholder={`Search ${title.toLowerCase()}...`}
        />
      </div>

      {/* Error state */}
      {error.value && (
        <div class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error.value}
        </div>
      )}

      {/* Table */}
      <div class="rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800">
        <DataTable
          columns={columns}
          data={displayed.value}
          sortKey={sortKey.value}
          sortDir={sortDir.value}
          onSort={handleSort}
          rowKey={(r) => r.metadata.uid}
          emptyMessage={loading.value
            ? "Loading resources..."
            : `No ${title.toLowerCase()} found`}
        />
      </div>
    </div>
  );
}
