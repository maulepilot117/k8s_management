import { Input } from "@/components/ui/Input.tsx";
import { Select } from "@/components/ui/Select.tsx";

interface PortEntry {
  name: string;
  port: number;
  targetPort: number;
  protocol: string;
  nodePort: number;
}

interface SelectorEntry {
  key: string;
  value: string;
}

interface ServicePortsProps {
  ports: PortEntry[];
  selector: SelectorEntry[];
  serviceType: string;
  errors: Record<string, string>;
  onChange: (field: string, value: unknown) => void;
}

const PROTOCOL_OPTIONS = [
  { value: "TCP", label: "TCP" },
  { value: "UDP", label: "UDP" },
];

export function ServicePortsStep({
  ports,
  selector,
  serviceType,
  errors,
  onChange,
}: ServicePortsProps) {
  const showNodePort = serviceType === "NodePort" ||
    serviceType === "LoadBalancer";

  const updatePort = (
    index: number,
    field: keyof PortEntry,
    val: string | number,
  ) => {
    const updated = [...ports];
    updated[index] = { ...updated[index], [field]: val };
    onChange("ports", updated);
  };

  const addPort = () => {
    if (ports.length >= 20) return;
    onChange("ports", [
      ...ports,
      { name: "", port: 0, targetPort: 0, protocol: "TCP", nodePort: 0 },
    ]);
  };

  const removePort = (index: number) => {
    onChange("ports", ports.filter((_, i) => i !== index));
  };

  const updateSelector = (
    index: number,
    field: "key" | "value",
    val: string,
  ) => {
    const updated = [...selector];
    updated[index] = { ...updated[index], [field]: val };
    onChange("selector", updated);
  };

  const addSelector = () => {
    onChange("selector", [...selector, { key: "", value: "" }]);
  };

  const removeSelector = (index: number) => {
    onChange("selector", selector.filter((_, i) => i !== index));
  };

  return (
    <div class="space-y-8 max-w-2xl">
      {/* Selector */}
      <div class="space-y-3">
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300">
            Pod Selector
          </label>
          <p class="text-xs text-slate-500 mt-1">
            Labels used to match target pods. Must match at least one pod's
            labels.
          </p>
        </div>
        {errors.selector && <p class="text-sm text-danger">{errors.selector}
        </p>}
        {selector.map((s, i) => (
          <div key={i} class="flex items-center gap-2">
            <Input
              value={s.key}
              onInput={(e) =>
                updateSelector(
                  i,
                  "key",
                  (e.target as HTMLInputElement).value,
                )}
              placeholder="key"
              class="flex-1"
            />
            <span class="text-slate-400">=</span>
            <Input
              value={s.value}
              onInput={(e) =>
                updateSelector(
                  i,
                  "value",
                  (e.target as HTMLInputElement).value,
                )}
              placeholder="value"
              class="flex-1"
            />
            <button
              type="button"
              onClick={() => removeSelector(i)}
              class="p-1 text-slate-400 hover:text-danger"
              title="Remove selector"
            >
              <svg
                class="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={addSelector}
          class="text-sm text-brand hover:text-brand/80"
        >
          + Add Selector
        </button>
      </div>

      {/* Ports */}
      <div class="space-y-3">
        <label class="block text-sm font-medium text-slate-700 dark:text-slate-300">
          Service Ports
        </label>
        {errors.ports && <p class="text-sm text-danger">{errors.ports}</p>}
        {ports.map((port, i) => (
          <div key={i} class="flex items-end gap-2">
            <div class="w-24">
              <Input
                label={i === 0 ? "Name" : undefined}
                value={port.name}
                onInput={(e) =>
                  updatePort(i, "name", (e.target as HTMLInputElement).value)}
                placeholder="http"
              />
            </div>
            <div class="w-20">
              <Input
                label={i === 0 ? "Port" : undefined}
                type="number"
                value={port.port ? String(port.port) : ""}
                onInput={(e) =>
                  updatePort(
                    i,
                    "port",
                    parseInt((e.target as HTMLInputElement).value) || 0,
                  )}
                placeholder="80"
                min={1}
                max={65535}
                error={errors[`ports[${i}].port`]}
              />
            </div>
            <div class="w-24">
              <Input
                label={i === 0 ? "Target" : undefined}
                type="number"
                value={port.targetPort ? String(port.targetPort) : ""}
                onInput={(e) =>
                  updatePort(
                    i,
                    "targetPort",
                    parseInt((e.target as HTMLInputElement).value) || 0,
                  )}
                placeholder="8080"
                min={1}
                max={65535}
                error={errors[`ports[${i}].targetPort`]}
              />
            </div>
            <div class="w-20">
              <Select
                label={i === 0 ? "Proto" : undefined}
                value={port.protocol}
                onChange={(e) =>
                  updatePort(
                    i,
                    "protocol",
                    (e.target as HTMLSelectElement).value,
                  )}
                options={PROTOCOL_OPTIONS}
              />
            </div>
            {showNodePort && (
              <div class="w-24">
                <Input
                  label={i === 0 ? "NodePort" : undefined}
                  type="number"
                  value={port.nodePort ? String(port.nodePort) : ""}
                  onInput={(e) =>
                    updatePort(
                      i,
                      "nodePort",
                      parseInt((e.target as HTMLInputElement).value) || 0,
                    )}
                  placeholder="30080"
                  min={30000}
                  max={32767}
                  error={errors[`ports[${i}].nodePort`]}
                />
              </div>
            )}
            <button
              type="button"
              onClick={() => removePort(i)}
              class="p-2 text-slate-400 hover:text-danger mb-1"
              title="Remove port"
            >
              <svg
                class="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        ))}
        {ports.length < 20 && (
          <button
            type="button"
            onClick={addPort}
            class="text-sm text-brand hover:text-brand/80"
          >
            + Add Port
          </button>
        )}
      </div>
    </div>
  );
}
