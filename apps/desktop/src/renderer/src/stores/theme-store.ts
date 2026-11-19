import { AppSettingTheme } from '@croffledev/common';
import { defineStore } from 'pinia';
import { onMounted, ref } from 'vue';

const resolveIsDark = (theme: AppSettingTheme): boolean => {
  if (theme === AppSettingTheme.DARK) {
    return true;
  }
  if (theme === AppSettingTheme.LIGHT) {
    return false;
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
};

export const useThemeStore = defineStore('darkTheme', () => {
  const isDark = ref<boolean>(false);
  const currentTheme = ref<AppSettingTheme>(AppSettingTheme.SYSTEM);

  const applyDomTheme = (dark: boolean) => {
    document.documentElement.classList.toggle('dark', dark);
    localStorage.setItem('theme', dark ? 'dark' : 'light');
    isDark.value = dark;
  };

  const applyFromSettings = (theme: AppSettingTheme) => {
    currentTheme.value = theme;
    applyDomTheme(resolveIsDark(theme));
  };

  onMounted(() => {
    const stored = localStorage.getItem('theme');
    if (stored) {
      isDark.value = stored === 'dark';
      document.documentElement.classList.toggle('dark', isDark.value);
    }
  });

  return {
    isDark,
    currentTheme,
    applyFromSettings,
  };
});
