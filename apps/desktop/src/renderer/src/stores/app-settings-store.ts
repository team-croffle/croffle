import {
  type AppSettingTheme,
  lookOf,
  type LookPreview,
  type StoredSettings,
} from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import { defineStore } from 'pinia';
import { ref } from 'vue';

import { setI18nLocale } from '@/i18n';
import { croffleInternal } from '@/services/host-bridge';
import { applyAppearance, DEFAULT_ACCENT_HUE } from '@/utils/appearance';

import { useThemeStore } from './theme-store';

export const useAppSettingsStore = defineStore('appSettings', () => {
  const settings = ref<AppSettings | null>(null);
  const isReady = ref(false);
  /** The settings screen's theme or colour not stored yet. Shown over every stored value. */
  let preview: LookPreview = {};
  /** main's revision of `settings`; an older or equal one is ignored (see StoredSettings). */
  let revision = -1;
  let unsubscribe: (() => void) | null = null;

  /** Applies stored settings with the preview on top — the only place the look is applied. */
  const applyToUi = (value: AppSettings) => {
    const look = lookOf(
      {
        theme: value.general.theme,
        accentHue: value.appearance?.accentHue ?? DEFAULT_ACCENT_HUE,
      },
      preview,
    );
    useThemeStore().applyFromSettings(look.theme);
    setI18nLocale(value.general.language);
    applyAppearance({ accentHue: look.accentHue });
  };

  const reapply = () => {
    if (settings.value) {
      applyToUi(settings.value);
    }
  };

  const receive = (stored: StoredSettings) => {
    if (stored.revision <= revision) {
      return;
    }
    revision = stored.revision;
    settings.value = stored.settings;
    applyToUi(stored.settings);
  };

  const load = async () => {
    receive(await croffleInternal.settings.getStored());
    isReady.value = true;
  };

  const initialize = async () => {
    unsubscribe?.();
    // Subscribe first so nothing stored between the read and the subscription is missed.
    unsubscribe = croffleInternal.settings.onStored(receive);
    await load();
  };

  const dispose = () => {
    unsubscribe?.();
    unsubscribe = null;
  };

  /** Shows a theme before it is stored (cleared by endPreview when its save ends). */
  const setThemeDraft = (theme: AppSettingTheme) => {
    preview.theme = theme;
    reapply();
  };

  /** Shows an accent hue before it is stored (cleared by endPreview when its save ends). */
  const setAccentHueDraft = (accentHue: number) => {
    preview.accentHue = accentHue;
    reapply();
  };

  /** The save of a previewed value ended (stored or failed): show what is stored. */
  const endPreview = (key: keyof LookPreview) => {
    delete preview[key];
    reapply();
  };

  /** Replaces the whole preview at once (one repaint); `{}` shows what is stored. */
  const setPreview = (next: LookPreview) => {
    preview = { ...next };
    reapply();
  };

  return {
    settings,
    isReady,
    initialize,
    dispose,
    load,
    setThemeDraft,
    setAccentHueDraft,
    endPreview,
    setPreview,
    receive,
  };
});
