import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function ConfigMapsPage() {
  return <ResourceTable kind="configmaps" title="ConfigMaps" />;
});
