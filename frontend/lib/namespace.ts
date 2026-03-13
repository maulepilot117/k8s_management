/**
 * Client-only module — MUST NOT be imported in server-rendered components.
 * Module-level signal is a process-global singleton in Deno; importing
 * this server-side would leak state across SSR requests.
 *
 * Shared namespace state consumed by TopBar (writes) and all resource
 * table islands (reads).
 */
import { signal } from "@preact/signals";

/** Currently selected namespace. "all" = all namespaces. */
export const selectedNamespace = signal<string>("all");
