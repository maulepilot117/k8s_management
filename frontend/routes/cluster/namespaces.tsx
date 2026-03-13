import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function NamespacesPage() {
  return <ResourceTable kind="namespaces" title="Namespaces" clusterScoped />;
});
