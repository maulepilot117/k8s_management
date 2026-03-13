import { SectionHeader } from "@/components/ui/Field.tsx";

interface Rule {
  apiGroups?: string[];
  resources?: string[];
  verbs?: string[];
}

export function RulesTable({ rules }: { rules?: Rule[] }) {
  if (!rules?.length) {
    return (
      <p class="text-sm text-slate-500 dark:text-slate-400">No rules defined</p>
    );
  }

  return (
    <div>
      <SectionHeader>Rules</SectionHeader>
      <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-slate-200 dark:border-slate-700">
              <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                API Groups
              </th>
              <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                Resources
              </th>
              <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                Verbs
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
            {rules.map((rule, i) => (
              <tr key={i}>
                <td class="px-3 py-1.5 font-mono text-xs text-slate-700 dark:text-slate-300">
                  {rule.apiGroups?.join(", ") || "*"}
                </td>
                <td class="px-3 py-1.5 font-mono text-xs text-slate-700 dark:text-slate-300">
                  {rule.resources?.join(", ") || "*"}
                </td>
                <td class="px-3 py-1.5 font-mono text-xs text-slate-700 dark:text-slate-300">
                  {rule.verbs?.join(", ") || "*"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
