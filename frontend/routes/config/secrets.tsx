import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function SecretsPage() {
  return <ResourceTable kind="secrets" title="Secrets" enableWS={false} />;
});
