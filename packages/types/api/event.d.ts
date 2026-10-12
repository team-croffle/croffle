/**
 * App-wide event bus.
 *
 * From an extension, `emit(name)` is namespaced with the extension id: `emit('ping')`
 * is delivered as `<extensionId>:ping` (a name already carrying that prefix is kept).
 * Core events (`AppEventType`) and other extensions' prefixes cannot be emitted; they
 * are dropped with a console warning. `on` subscribes to any name, core events included.
 */
export interface EventApi {
  emit(eventName: string, ...args: unknown[]): void;
  on(eventName: string, callback: (...args: unknown[]) => void): () => void;
}
