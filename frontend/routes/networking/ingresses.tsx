import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function IngressesPage() {
  return <ResourceTable kind="ingresses" title="Ingresses" />;
});
