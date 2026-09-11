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

  /**
   * 타이틀 바 빠른 전환 (light ↔ dark).
   * 설정값(currentTheme)이 아니라 실제 표시 상태(isDark)를 기준으로 뒤집는다.
   * SYSTEM 설정에서 OS가 이미 다크일 때도 첫 클릭에 바로 바뀌도록 하기 위함.
   * 설정(general.theme)은 바꾸지 않는 세션 임시 오버라이드.
   */
  const changeTheme = (): void => {
    const next = isDark.value ? AppSettingTheme.LIGHT : AppSettingTheme.DARK;
    applyFromSettings(next);
  };

  return {
    isDark,
    currentTheme,
    applyFromSettings,
    changeTheme,
  };
});
