import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function ServicesPage() {
  return (
    <ResourceTable
      kind="services"
      title="Services"
      createHref="/networking/services/new"
    />
  );
});
