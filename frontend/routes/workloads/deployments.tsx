import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function DeploymentsPage() {
  return (
    <ResourceTable
      kind="deployments"
      title="Deployments"
      createHref="/workloads/deployments/new"
    />
  );
});
