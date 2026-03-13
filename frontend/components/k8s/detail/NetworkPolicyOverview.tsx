import type { K8sResource, NetworkPolicy } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";
import { KeyValueTable } from "./KeyValueTable.tsx";

export function NetworkPolicyOverview({ resource }: { resource: K8sResource }) {
  const np = resource as NetworkPolicy;
  const spec = np.spec;

  const ingressCount = spec?.ingress?.length ?? 0;
  const egressCount = spec?.egress?.length ?? 0;

  return (
    <div class="space-y-4">
      {/* Summary */}
      <div>
        <SectionHeader>Summary</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field
            label="Policy Types"
            value={spec?.policyTypes?.join(", ") ?? "Ingress"}
          />
          <Field label="Ingress Rules" value={String(ingressCount)} />
          <Field label="Egress Rules" value={String(egressCount)} />
        </div>
      </div>

      {/* Pod Selector */}
      {spec?.podSelector?.matchLabels &&
        Object.keys(spec.podSelector.matchLabels).length > 0 && (
        <KeyValueTable
          title="Pod Selector"
          data={spec.podSelector.matchLabels}
        />
      )}
    </div>
  );
}
