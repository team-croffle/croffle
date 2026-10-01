import type { ExtensionInfo } from '@croffledev/common';
import type { ExtensionContext, RegisterConfigurationTabOptions } from '@croffledev/croffle-types';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';

function t(key: string, values?: Record<string, unknown>) {
  return String(values ? i18n.global.t(key, values) : i18n.global.t(key));
}

class ExtensionLoader {
  private activeExtensions = new Map<string, unknown>();

  public async init() {
    try {
      const enabled = await croffle.extensions.info.getEnabled();
      for (const extension of enabled) {
        await this.loadExtension(extension);
      }
    } catch {
      toast.error(t('extensions.loadFailed'));
    }
  }

  public async loadPluginById(extensionId: string) {
    try {
      const installed = await croffle.extensions.info.getInstalled();
      const extension = installed.find((e) => e.id === extensionId);
      if (extension?.enabled) {
        await this.loadExtension(extension);
      }
    } catch {
      toast.error(t('extensions.loadOneFailed', { id: extensionId }));
    }
  }

  private async loadExtension(extension: ExtensionInfo) {
    try {
      if (this.activeExtensions.has(extension.id)) {
        return;
      }

      try {
        const styleUrl = `extension://${extension.id}/style.css`;
        const res = await fetch(styleUrl, { method: 'HEAD' });
        if (res.ok) {
          const link = document.createElement('link');
          link.rel = 'stylesheet';
          link.href = `${styleUrl}?t=${Date.now()}`;
          link.id = `extension-style-${extension.id}`;
          document.head.appendChild(link);
        }
      } catch {
        // style.css is optional
      }

      if (extension.main) {
        const entryUrl = `extension://${extension.id}/${extension.main}?t=${Date.now()}`;
        const mod = await import(/* @vite-ignore */ entryUrl);

        if (mod && typeof mod.activated === 'function') {
          const context = this.createContext(extension);
          await mod.activated(context);
          this.activeExtensions.set(extension.id, { module: mod, context });
        }
      } else {
        this.activeExtensions.set(extension.id, { module: null, context: null });
      }

      window.dispatchEvent(
        new CustomEvent('extension:loaded', {
          detail: { extension },
        }),
      );
    } catch {
      toast.error(t('extensions.executeFailed', { name: extension.name }));
    }
  }

  public async unloadPlugin(extensionId: string) {
    try {
      const active = this.activeExtensions.get(extensionId) as
        // oxlint-disable-next-line typescript/no-explicit-any
        { module: any; context: ExtensionContext } | undefined;
      if (active) {
        if (active.module && typeof active.module.deactivated === 'function') {
          await active.module.deactivated(active.context);
        }
        this.activeExtensions.delete(extensionId);

        const link = document.getElementById(`extension-style-${extensionId}`);
        link?.remove();

        window.dispatchEvent(
          new CustomEvent('extension:unloaded', {
            detail: { extensionId },
          }),
        );
      }
    } catch {
      toast.error(t('extensions.unloadFailed', { id: extensionId }));
    }
  }

  private createContext(extension: ExtensionInfo): ExtensionContext {
    const id = extension.id;

    return {
      ...croffle,
      storage: {
        get: (key) => croffle.extensions.storage.get(id, key),
        set: (key, value) => croffle.extensions.storage.set(id, key, value),
        delete: (key) => croffle.extensions.storage.delete(id, key),
        clear: () => croffle.extensions.storage.clear(id),
      },
      session: {
        get: (key) => croffle.extensions.session.get(id, key),
        set: (key, value) => croffle.extensions.session.set(id, key, value),
        delete: (key) => croffle.extensions.session.delete(id, key),
        clear: () => croffle.extensions.session.clear(id),
      },
      configuration: {
        get: (storageKey) => croffle.extensions.configuration.get(id, storageKey),
        set: (values, storageKey) => croffle.extensions.configuration.set(id, values, storageKey),
      },
      ui: {
        registerView: (viewId, renderFn) => {
          window.dispatchEvent(
            new CustomEvent('extension:register-view', {
              detail: { extensionId: extension.id, viewId, renderFn },
            }),
          );
        },
        registerContextMenu: (target, command, label, callback) => {
          window.dispatchEvent(
            new CustomEvent('extension:register-context-menu', {
              detail: { extensionId: extension.id, target, command, label, callback },
            }),
          );
        },
        onContextMenu: (menuId, handler, options) => {
          const detail = {
            extensionId: extension.id,
            menuId,
            handler,
            condition: options?.condition,
          };
          window.dispatchEvent(new CustomEvent('extension:bind-context-menu', { detail }));
          return () => {
            window.dispatchEvent(new CustomEvent('extension:unbind-context-menu', { detail }));
          };
        },
        registerConfigurationTab: (tabId: string, options: RegisterConfigurationTabOptions) => {
          window.dispatchEvent(
            new CustomEvent('extension:register-configuration-tab', {
              detail: {
                extensionId: extension.id,
                extensionName: extension.name,
                tabId,
                ...options,
              },
            }),
          );
        },
      },
    };
  }
}

export const extensionLoader = new ExtensionLoader();
