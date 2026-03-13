import type { K8sResource, Secret } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";

export function SecretOverview({ resource }: { resource: K8sResource }) {
  const s = resource as Secret;
  const entries = Object.entries(s.data ?? {}).sort(([a], [b]) =>
    a.localeCompare(b)
  );

  return (
    <div class="space-y-4">
      {/* Type */}
      <div>
        <SectionHeader>Summary</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Type" value={s.type ?? "Opaque"} mono />
          <Field label="Keys" value={String(entries.length)} />
        </div>
      </div>

      {/* Data Keys (masked) */}
      {entries.length > 0 && (
        <div>
          <SectionHeader>Data</SectionHeader>
          <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-slate-200 dark:border-slate-700">
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Key
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Value
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
                {entries.map(([key]) => (
                  <tr key={key}>
                    <td class="px-3 py-1.5 font-mono text-xs text-cyan-700 dark:text-cyan-400 whitespace-nowrap">
                      {key}
                    </td>
                    <td class="px-3 py-1.5 font-mono text-xs text-slate-400 dark:text-slate-500">
                      ****
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
