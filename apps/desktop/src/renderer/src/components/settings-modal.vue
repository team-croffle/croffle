<script setup lang="ts">
  import {
    AppCloseBehavior,
    AppSettingLanguage,
    AppSettingTheme,
    CalendarTimeFormat,
    CalendarView,
    CalendarWeekStartDay,
  } from '@croffledev/common';
  import type { ConfigurationTabContribution, ExtensionInfo } from '@croffledev/common';
  import type { AppSettings } from '@croffledev/croffle-types';
  import { ref, onMounted, watch, computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { toast } from 'vue-sonner';

  import ConfigSchemaForm from '@/components/settings/config-schema-form.vue';
  import SettingsExtensionPanel from '@/components/settings/settings-extension-panel.vue';
  import { Button } from '@/components/ui/button';
  import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
  } from '@/components/ui/dialog';
  import { Icon } from '@/components/ui/icon';
  import { Label } from '@/components/ui/label';
  import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
  } from '@/components/ui/select';
  import { Separator } from '@/components/ui/separator';
  import { Switch } from '@/components/ui/switch';
  import { extensionLoader } from '@/services/extension-loader';
  import { useAppSettingsStore } from '@/stores/app-settings-store';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useSettingsStore } from '@/stores/settings-store';
  import {
    DEFAULT_ACCENT_HUE,
    DEFAULT_ACCENT_PRESET_HEX,
    hexToHue,
    hueToPreviewHex,
    huesMatch,
  } from '@/utils/appearance';
  import { mergeWithSchemaDefaults } from '@/utils/extension-configuration-schema';

  type Props = {
    open: boolean;
  };

  const props = defineProps<Props>();
  const emit = defineEmits<{
    (e: 'update:open', value: boolean): void;
  }>();

  const settingsStore = useSettingsStore();
  const appSettingsStore = useAppSettingsStore();
  const scheduleStore = useScheduleStore();
  const { t } = useI18n();

  const activeTab = ref<string>('general');
  const originalSettings = ref<AppSettings | null>(null);
  const settings = ref<AppSettings | null>(null);

  // 로딩/오류 상태
  const isLoadingSettings = ref(true);
  const loadError = ref<string | null>(null);

  // 플러그인 관리 상태
  const installedPlugins = ref<ExtensionInfo[]>([]);
  const installUrl = ref<string>('');
  const isInstalling = ref<boolean>(false);

  // 일정 데이터 가져오기/내보내기 (설정 저장과 무관한 즉시 실행)
  type ImportMode = 'merge' | 'duplicate';
  const importMode = ref<ImportMode>('merge');
  const isExporting = ref(false);
  const isImporting = ref(false);

  const errorMessage = (error: unknown) =>
    error instanceof Error ? error.message : typeof error === 'string' ? error : String(error);

  const extensionDrafts = ref<Record<string, Record<string, unknown>>>({});
  const originalExtensionDrafts = ref<Record<string, Record<string, unknown>>>({});

  /** Croffle main first, then a small rainbow of accent options. */
  const ACCENT_COLOR_PRESETS = [
    DEFAULT_ACCENT_PRESET_HEX,
    '#F87171',
    '#FBBF24',
    '#34D399',
    '#60A5FA',
    '#A78BFA',
    '#F472B6',
  ];

  const builtinTabs = computed(() => [
    { id: 'general', label: t('settings.tabs.general'), icon: 'lucide:settings' },
    { id: 'calendar', label: t('settings.tabs.calendar'), icon: 'lucide:calendar-days' },
    { id: 'notifications', label: t('settings.tabs.notifications'), icon: 'lucide:bell' },
    { id: 'extensions', label: t('settings.tabs.extensions'), icon: 'lucide:puzzle' },
  ]);

  const isBootEnabled = computed(() => settings.value?.general.startOnSystemBoot ?? false);

  const currentAccentHue = computed(
    () => settings.value?.appearance?.accentHue ?? DEFAULT_ACCENT_HUE,
  );

  const accentPreviewHex = computed(() => hueToPreviewHex(currentAccentHue.value));

  const allTabs = computed(() => {
    const extension = settingsStore.sortedExtensionTabs.map((tab) => ({
      id: settingsStore.getTabCompositeId(tab),
      label: tab.label,
      icon: tab.icon ?? 'lucide:puzzle',
      extensionTab: tab,
    }));
    return [...builtinTabs.value, ...extension];
  });

  const activeExtensionTab = computed(() => settingsStore.findExtensionTab(activeTab.value));

  const activeExtensionDraft = computed({
    get: () => extensionDrafts.value[activeTab.value] ?? {},
    set: (value: Record<string, unknown>) => {
      extensionDrafts.value[activeTab.value] = value;
    },
  });

  const activeTabLabel = computed(
    () => allTabs.value.find((tab) => tab.id === activeTab.value)?.label ?? t('settings.title'),
  );

  // 깊은 복사 유틸
  const cloneSettings = (value: AppSettings) => JSON.parse(JSON.stringify(value)) as AppSettings;
  const cloneExtensionDrafts = (value: Record<string, Record<string, unknown>>) =>
    JSON.parse(JSON.stringify(value)) as Record<string, Record<string, unknown>>;

  const loadExtensionTabSettings = async (tab: ConfigurationTabContribution) => {
    if (!tab.sections?.length) {
      return;
    }
    const compositeId = settingsStore.getTabCompositeId(tab);
    const stored = await croffle.extensions.configuration.get<Record<string, unknown>>(
      tab.extensionId,
    );
    const merged = mergeWithSchemaDefaults(tab.sections, stored);
    extensionDrafts.value[compositeId] = merged;
    originalExtensionDrafts.value[compositeId] = JSON.parse(JSON.stringify(merged));
  };

  const loadAllExtensionSettings = async () => {
    const drafts: Record<string, Record<string, unknown>> = {};
    const originals: Record<string, Record<string, unknown>> = {};

    for (const tab of settingsStore.sortedExtensionTabs) {
      if (!tab.sections?.length) {
        continue;
      }
      const compositeId = settingsStore.getTabCompositeId(tab);
      const stored = await croffle.extensions.configuration.get<Record<string, unknown>>(
        tab.extensionId,
      );
      const merged = mergeWithSchemaDefaults(tab.sections, stored);
      drafts[compositeId] = merged;
      originals[compositeId] = JSON.parse(JSON.stringify(merged));
    }

    extensionDrafts.value = drafts;
    originalExtensionDrafts.value = originals;
  };

  const reloadSettings = async () => {
    isLoadingSettings.value = true;
    loadError.value = null;
    try {
      const loaded = await croffle.settings.getAll();
      originalSettings.value = loaded;
      settings.value = cloneSettings(loaded);
      await loadAllExtensionSettings();
    } catch (error) {
      toast.error(t('settings.errors.load', { error: JSON.stringify(error) }));
      loadError.value = t('settings.loadFailed');
    } finally {
      isLoadingSettings.value = false;
    }
  };

  const fetchInstalledPlugins = async () => {
    try {
      installedPlugins.value = await croffle.extensions.info.getInstalled();
    } catch (err) {
      toast.error(t('settings.errors.extensionsList', { error: JSON.stringify(err) }));
    }
  };

  const onInstallPlugin = async () => {
    if (!installUrl.value) {
      return;
    }
    isInstalling.value = true;
    try {
      const plugin = await croffle.extensions.info.install({
        id: installUrl.value,
      });
      installUrl.value = '';
      await fetchInstalledPlugins();
      await extensionLoader.loadPluginById(plugin.id);
    } catch (err) {
      toast.error(t('settings.errors.install', { error: JSON.stringify(err) }));
    } finally {
      isInstalling.value = false;
    }
  };

  const onLocalZipSelect = async () => {
    isInstalling.value = true;
    try {
      const result = await croffle.extensions.info.installFromLocal();
      if (result) {
        await fetchInstalledPlugins();
        await extensionLoader.loadPluginById(result.id);
      }
    } catch (err) {
      toast.error(t('settings.errors.installLocal', { error: JSON.stringify(err) }));
    } finally {
      isInstalling.value = false;
    }
  };

  const onTogglePlugin = async (plugin: ExtensionInfo) => {
    try {
      await croffle.extensions.info.toggle(plugin.id, plugin.enabled);
      await fetchInstalledPlugins();

      if (plugin.enabled) {
        await extensionLoader.loadPluginById(plugin.id);
      } else {
        await extensionLoader.unloadPlugin(plugin.id);
      }
    } catch (err) {
      toast.error(t('settings.errors.toggle', { error: JSON.stringify(err) }));
    }
  };

  const onUninstallExtension = async (plugin: ExtensionInfo) => {
    if (!confirm(t('settings.extensions.uninstallConfirm', { name: plugin.name }))) {
      return;
    }
    try {
      await extensionLoader.unloadPlugin(plugin.id);
      await croffle.extensions.info.uninstall(plugin.id);
      await fetchInstalledPlugins();
    } catch (err) {
      toast.error(t('settings.errors.uninstall', { error: JSON.stringify(err) }));
    }
  };

  onMounted(() => {
    void reloadSettings();
    void fetchInstalledPlugins();
  });

  watch(
    () => props.open,
    (isOpen) => {
      if (!isOpen) {
        return;
      }
      if (originalSettings.value) {
        settings.value = cloneSettings(originalSettings.value);
        reapplyPersistedAppearance(originalSettings.value);
        extensionDrafts.value = cloneExtensionDrafts(originalExtensionDrafts.value);
      } else {
        void reloadSettings();
      }
      if (activeTab.value === 'extensions') {
        void fetchInstalledPlugins();
      }
    },
  );

  watch(activeTab, (tab) => {
    if (tab === 'extensions') {
      void fetchInstalledPlugins();
    }
  });

  const handleSave = async () => {
    if (!settings.value) {
      return;
    }
    try {
      await croffle.settings.update(cloneSettings(settings.value));

      for (const tab of settingsStore.sortedExtensionTabs) {
        if (!tab.sections?.length) {
          continue;
        }
        const compositeId = settingsStore.getTabCompositeId(tab);
        const draft = extensionDrafts.value[compositeId];
        if (draft) {
          await croffle.extensions.configuration.set(tab.extensionId, draft);
        }
      }

      // 저장 후 재조회(실제 반영값 동기화)
      const reloaded = await croffle.settings.getAll();
      originalSettings.value = reloaded;
      settings.value = cloneSettings(reloaded);

      originalExtensionDrafts.value = cloneExtensionDrafts(extensionDrafts.value);

      emit('update:open', false);
    } catch (error) {
      toast.error(t('settings.errors.save', { error: JSON.stringify(error) }));
    }
  };

  const reapplyPersistedAppearance = (value: AppSettings) => {
    appSettingsStore.setThemeDraft(value.general.theme);
    appSettingsStore.setAccentHueDraft(value.appearance?.accentHue ?? DEFAULT_ACCENT_HUE);
  };

  const handleCancel = () => {
    if (originalSettings.value) {
      settings.value = cloneSettings(originalSettings.value);
      reapplyPersistedAppearance(originalSettings.value);
    }
    extensionDrafts.value = cloneExtensionDrafts(originalExtensionDrafts.value);

    emit('update:open', false);
  };

  // Switch 이벤트값 정규화(checked/modelValue 둘 다 대응)
  const asBool = (v: unknown): boolean => {
    if (typeof v === 'boolean') {
      return v;
    }
    if (typeof v === 'string') {
      return v === 'true' || v === '1' || v === 'on';
    }
    return !!v;
  };

  const setGeneralBool = (
    key: 'autoUpdate' | 'startOnSystemBoot' | 'startMinimized',
    v: unknown,
  ) => {
    if (!settings.value) {
      return;
    }
    settings.value.general[key] = asBool(v);
  };

  const onThemeChange = (theme: unknown) => {
    if (!settings.value || typeof theme !== 'string') {
      return;
    }
    settings.value.general.theme = theme as AppSettingTheme;
    appSettingsStore.setThemeDraft(theme as AppSettingTheme);
  };

  const ensureAppearance = () => {
    if (!settings.value) {
      return null;
    }
    if (!settings.value.appearance) {
      settings.value.appearance = { accentHue: DEFAULT_ACCENT_HUE };
    }
    return settings.value;
  };

  const onAccentHexChange = (hex: string) => {
    const draft = ensureAppearance();
    if (!draft) {
      return;
    }
    // First preset is the exact Croffle primary hue (avoid round-trip drift).
    const hue =
      hex.toUpperCase() === DEFAULT_ACCENT_PRESET_HEX.toUpperCase()
        ? DEFAULT_ACCENT_HUE
        : hexToHue(hex);
    draft.appearance.accentHue = hue;
    appSettingsStore.setAccentHueDraft(hue);
  };

  const onAccentColorInput = (event: Event) => {
    const target = event.target as HTMLInputElement | null;
    if (!target?.value) {
      return;
    }
    onAccentHexChange(target.value);
  };

  const isAccentPresetSelected = (hex: string) => huesMatch(hexToHue(hex), currentAccentHue.value);

  const onReminderMinutesChange = (v: unknown) => {
    if (!settings.value || v === null) {
      return;
    }
    settings.value.notifications.defaultReminderMinutes = Number(v);
  };

  const onAppAlertSwitch = (v: unknown) => {
    if (!settings.value) {
      return;
    }
    settings.value.notifications.enabled = asBool(v);
  };

  const onShowWeekNumbersSwitch = (v: unknown) => {
    if (!settings.value) {
      return;
    }
    settings.value.calendar.showWeekNumbers = asBool(v);
  };

  const languageOptions = computed(() => [
    { value: AppSettingLanguage.KO, label: t('language.ko') },
    { value: AppSettingLanguage.EN, label: t('language.en') },
  ]);

  const closeBehaviorOptions = computed(() => [
    { value: AppCloseBehavior.ASK, label: t('settings.general.closeBehaviorAsk') },
    { value: AppCloseBehavior.TRAY, label: t('settings.general.closeBehaviorTray') },
    { value: AppCloseBehavior.QUIT, label: t('settings.general.closeBehaviorQuit') },
  ]);

  const onCloseBehaviorChange = (value: unknown) => {
    if (!settings.value || typeof value !== 'string') {
      return;
    }
    settings.value.general.closeBehavior = value as AppCloseBehavior;
  };

  const themeOptions = computed(() => [
    { value: AppSettingTheme.LIGHT, label: t('settings.general.themeLight') },
    { value: AppSettingTheme.DARK, label: t('settings.general.themeDark') },
    { value: AppSettingTheme.SYSTEM, label: t('settings.general.themeSystem') },
  ]);

  const calendarViewOptions = computed(() => [
    { value: CalendarView.DAY, label: t('settings.calendar.viewDay') },
    { value: CalendarView.WEEK, label: t('settings.calendar.viewWeek') },
    { value: CalendarView.MONTH, label: t('settings.calendar.viewMonth') },
    { value: CalendarView.YEAR, label: t('settings.calendar.viewYear') },
  ]);

  const weekStartOptions = computed(() => [
    { value: CalendarWeekStartDay.SUNDAY, label: t('settings.calendar.weekStartSunday') },
    { value: CalendarWeekStartDay.MONDAY, label: t('settings.calendar.weekStartMonday') },
  ]);

  const timeFormatOptions = computed(() => [
    { value: CalendarTimeFormat.H12, label: t('settings.calendar.timeFormat12h') },
    { value: CalendarTimeFormat.H24, label: t('settings.calendar.timeFormat24h') },
  ]);

  const reminderMinuteOptions = computed(() =>
    [5, 10, 15, 30, 60].map((minutes) => ({
      value: String(minutes),
      label: t('settings.notifications.minutesBefore', { minutes }),
    })),
  );

  const importModeOptions = computed(() => [
    { value: 'merge' as const, label: t('settings.calendar.importMerge') },
    { value: 'duplicate' as const, label: t('settings.calendar.importDuplicate') },
  ]);

  const importModeHint = computed(() =>
    importMode.value === 'merge'
      ? t('settings.calendar.importMergeHint')
      : t('settings.calendar.importDuplicateHint'),
  );

  const onImportModeChange = (value: unknown) => {
    if (value === 'merge' || value === 'duplicate') {
      importMode.value = value;
    }
  };

  const onExportSchedules = async () => {
    if (isExporting.value) {
      return;
    }
    isExporting.value = true;
    try {
      const result = await croffle.calendar.schedules.exportSchedulesToFile();
      if (result) {
        toast.success(
          t('settings.calendar.exportDone', { count: result.count, path: result.filePath }),
        );
      }
    } catch (error) {
      toast.error(t('settings.errors.export', { error: errorMessage(error) }));
    } finally {
      isExporting.value = false;
    }
  };

  const onImportSchedules = async () => {
    if (isImporting.value) {
      return;
    }
    isImporting.value = true;
    try {
      const result = await croffle.calendar.schedules.importScheduleFromFile(importMode.value);
      if (result) {
        toast.success(
          t('settings.calendar.importDone', { created: result.created, updated: result.updated }),
        );
        await scheduleStore.reload();
      }
    } catch (error) {
      toast.error(t('settings.errors.import', { error: errorMessage(error) }));
      // 일부만 들어갔을 수 있으므로 화면은 항상 다시 읽는다
      await scheduleStore.reload();
    } finally {
      isImporting.value = false;
    }
  };

  const onExtensionTabClick = async (tab: ConfigurationTabContribution) => {
    const compositeId = settingsStore.getTabCompositeId(tab);
    activeTab.value = compositeId;
    if (tab.sections?.length && !extensionDrafts.value[compositeId]) {
      await loadExtensionTabSettings(tab);
    }
  };
</script>

<template>
  <Dialog :open="open" @update:open="(val) => emit('update:open', val)">
    <!-- 반응형 모달 크기 -->
    <DialogContent
      class="h-[90vh] max-h-235 w-[96vw] gap-0 overflow-hidden border-none p-0 shadow-2xl sm:w-[94vw] sm:max-w-none lg:w-[92vw]"
    >
      <DialogHeader class="sr-only">
        <DialogTitle>{{ $t('settings.title') }}</DialogTitle>
        <DialogDescription>{{ $t('settings.description') }}</DialogDescription>
      </DialogHeader>

      <div class="bg-background text-foreground absolute inset-0 flex overflow-hidden">
        <!-- 좌측 탭 -->
        <div class="border-border bg-muted/20 w-60 shrink-0 overflow-y-auto border-r p-4">
          <h2 class="text-foreground mb-6 px-2 text-xl font-bold">{{ $t('settings.title') }}</h2>
          <nav class="space-y-1">
            <button
              v-for="tab in allTabs"
              :key="tab.id"
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'bg-croffle-sidebar-selected text-croffle-primary'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
              ]"
              @click="
                'extensionTab' in tab && tab.extensionTab
                  ? onExtensionTabClick(tab.extensionTab)
                  : (activeTab = tab.id)
              "
            >
              <Icon v-if="typeof tab.icon === 'string'" :icon="tab.icon" class="h-4 w-4" />
              <component v-else :is="tab.icon" class="h-4 w-4" />
              {{ tab.label }}
            </button>
          </nav>
        </div>

        <!-- 우측 콘텐츠 -->
        <div class="flex min-h-0 min-w-0 flex-1 flex-col">
          <div class="shrink-0 px-6 py-6 md:px-8">
            <h3 class="text-2xl font-bold wrap-break-word">
              {{ activeTabLabel }}
            </h3>
          </div>

          <div class="flex-1 overflow-y-auto px-6 pb-8 md:px-8">
            <div v-if="isLoadingSettings" class="w-full max-w-none space-y-3">
              <p class="text-muted-foreground text-sm">{{ $t('settings.loading') }}</p>
            </div>

            <div v-else-if="loadError" class="w-full max-w-none space-y-3">
              <p class="text-destructive text-sm">{{ loadError }}</p>
              <div class="flex gap-2">
                <Button type="button" variant="outline" @click="reloadSettings">
                  {{ $t('common.retry') }}
                </Button>
                <Button type="button" @click="emit('update:open', false)">
                  {{ $t('common.close') }}
                </Button>
              </div>
            </div>

            <div v-else-if="settings" class="w-full max-w-none space-y-8">
              <!-- 일반 -->
              <div v-if="activeTab === 'general'" class="space-y-8">
                <p class="text-muted-foreground text-sm">{{ $t('settings.general.intro') }}</p>

                <section class="space-y-4">
                  <h4 class="text-base font-bold text-foreground">
                    {{ $t('settings.general.display') }}
                  </h4>

                  <div class="space-y-2">
                    <Label for="settings-language" class="text-foreground text-sm font-medium">
                      {{ $t('settings.general.language') }}
                    </Label>
                    <Select v-model="settings.general.language">
                      <SelectTrigger id="settings-language" class="w-full">
                        <SelectValue :placeholder="$t('settings.general.languagePlaceholder')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem
                          v-for="option in languageOptions"
                          :key="option.value"
                          :value="option.value"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div class="space-y-2">
                    <Label for="settings-theme" class="text-foreground text-sm font-medium">
                      {{ $t('settings.general.theme') }}
                    </Label>
                    <Select
                      :model-value="settings.general.theme"
                      @update:model-value="onThemeChange"
                    >
                      <SelectTrigger id="settings-theme" class="w-full">
                        <SelectValue :placeholder="$t('settings.general.themePlaceholder')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem
                          v-for="option in themeOptions"
                          :key="option.value"
                          :value="option.value"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div class="space-y-2">
                    <Label class="text-foreground text-sm font-medium">
                      {{ $t('settings.general.accentColor') }}
                    </Label>
                    <div class="flex flex-wrap items-center gap-2">
                      <button
                        v-for="color in ACCENT_COLOR_PRESETS"
                        :key="color"
                        type="button"
                        class="size-8 cursor-pointer rounded-full border-2 transition-transform hover:scale-105"
                        :class="
                          isAccentPresetSelected(color)
                            ? 'border-foreground scale-105'
                            : 'border-transparent'
                        "
                        :style="{ backgroundColor: color }"
                        :aria-label="$t('settings.general.accentColorAria', { color })"
                        @click="onAccentHexChange(color)"
                      />
                      <label
                        class="border-croffle-border hover:bg-muted/50 relative flex size-8 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed"
                        :title="$t('settings.general.customAccentColor')"
                      >
                        <Icon icon="lucide:palette" class="text-muted-foreground size-3.5" />
                        <input
                          type="color"
                          class="absolute inset-0 cursor-pointer opacity-0"
                          :value="accentPreviewHex"
                          :aria-label="$t('settings.general.customAccentColorAria')"
                          @input="onAccentColorInput"
                        />
                      </label>
                    </div>
                    <p class="text-muted-foreground text-xs">
                      {{ $t('settings.general.accentColorHint') }}
                    </p>
                  </div>
                </section>

                <Separator />

                <section class="space-y-4">
                  <h4 class="text-base font-bold text-foreground">
                    {{ $t('settings.general.updates') }}
                  </h4>
                  <div class="flex items-center justify-between">
                    <div class="space-y-0.5">
                      <Label class="text-foreground text-sm font-medium">
                        {{ $t('settings.general.autoUpdate') }}
                      </Label>
                      <p class="text-muted-foreground text-xs">
                        {{ $t('settings.general.autoUpdateHint') }}
                      </p>
                    </div>
                    <Switch
                      :checked="settings.general.autoUpdate"
                      :model-value="settings.general.autoUpdate"
                      :aria-label="$t('settings.general.autoUpdate')"
                      @update:checked="(v: boolean) => setGeneralBool('autoUpdate', v)"
                      @update:model-value="(v: unknown) => setGeneralBool('autoUpdate', v)"
                    />
                  </div>
                </section>

                <Separator />

                <section class="space-y-4">
                  <h4 class="text-base font-bold text-foreground">
                    {{ $t('settings.general.startup') }}
                  </h4>

                  <div class="flex items-center justify-between">
                    <div class="space-y-0.5">
                      <Label class="text-foreground text-sm font-medium">
                        {{ $t('settings.general.startOnBoot') }}
                      </Label>
                      <p class="text-muted-foreground text-xs">
                        {{ $t('settings.general.startOnBootHint') }}
                      </p>
                    </div>
                    <Switch
                      :checked="settings.general.startOnSystemBoot"
                      :model-value="settings.general.startOnSystemBoot"
                      :aria-label="$t('settings.general.startOnBoot')"
                      @update:checked="(v: boolean) => setGeneralBool('startOnSystemBoot', v)"
                      @update:model-value="(v: unknown) => setGeneralBool('startOnSystemBoot', v)"
                    />
                  </div>

                  <div
                    class="space-y-4"
                    :class="{
                      'pointer-events-none opacity-50': !isBootEnabled,
                    }"
                  >
                    <div class="flex items-center justify-between">
                      <div class="space-y-0.5">
                        <Label class="text-foreground text-sm font-medium">
                          {{ $t('settings.general.startMinimized') }}
                        </Label>
                        <p class="text-muted-foreground text-xs">
                          {{ $t('settings.general.startMinimizedHint') }}
                        </p>
                      </div>
                      <Switch
                        :checked="settings.general.startMinimized"
                        :model-value="settings.general.startMinimized"
                        :disabled="!isBootEnabled"
                        :aria-label="$t('settings.general.startMinimized')"
                        @update:checked="(v: boolean) => setGeneralBool('startMinimized', v)"
                        @update:model-value="(v: unknown) => setGeneralBool('startMinimized', v)"
                      />
                    </div>
                  </div>
                </section>

                <Separator />

                <section class="space-y-4">
                  <h4 class="text-base font-bold text-foreground">
                    {{ $t('settings.general.window') }}
                  </h4>

                  <div class="space-y-2">
                    <Label
                      for="settings-close-behavior"
                      class="text-foreground text-sm font-medium"
                    >
                      {{ $t('settings.general.closeBehavior') }}
                    </Label>
                    <Select
                      :model-value="settings.general.closeBehavior"
                      @update:model-value="onCloseBehaviorChange"
                    >
                      <SelectTrigger id="settings-close-behavior" class="w-full">
                        <SelectValue
                          :placeholder="$t('settings.general.closeBehaviorPlaceholder')"
                        />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem
                          v-for="option in closeBehaviorOptions"
                          :key="option.value"
                          :value="option.value"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <p class="text-muted-foreground text-xs">
                      {{ $t('settings.general.closeBehaviorHint') }}
                    </p>
                  </div>
                </section>
              </div>

              <!-- 캘린더 -->
              <div v-if="activeTab === 'calendar'" class="space-y-6">
                <p class="text-muted-foreground text-sm">{{ $t('settings.calendar.intro') }}</p>

                <div class="space-y-2">
                  <Label for="settings-default-view" class="text-foreground text-sm font-medium">
                    {{ $t('settings.calendar.defaultView') }}
                  </Label>
                  <Select v-model="settings.calendar.defaultView">
                    <SelectTrigger id="settings-default-view" class="w-full">
                      <SelectValue :placeholder="$t('settings.calendar.defaultViewPlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem
                        v-for="option in calendarViewOptions"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label for="settings-week-start" class="text-foreground text-sm font-medium">
                    {{ $t('settings.calendar.weekStart') }}
                  </Label>
                  <Select v-model="settings.calendar.weekStartDay">
                    <SelectTrigger id="settings-week-start" class="w-full">
                      <SelectValue :placeholder="$t('settings.calendar.weekStartPlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem
                        v-for="option in weekStartOptions"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="space-y-2">
                  <Label for="settings-time-format" class="text-foreground text-sm font-medium">
                    {{ $t('settings.calendar.timeFormat') }}
                  </Label>
                  <Select v-model="settings.calendar.timeFormat">
                    <SelectTrigger id="settings-time-format" class="w-full">
                      <SelectValue :placeholder="$t('settings.calendar.timeFormatPlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem
                        v-for="option in timeFormatOptions"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div class="flex items-center justify-between">
                  <div class="space-y-0.5">
                    <Label class="text-foreground text-sm font-medium">
                      {{ $t('settings.calendar.showWeekNumbers') }}
                    </Label>
                    <p class="text-muted-foreground text-xs">
                      {{ $t('settings.calendar.showWeekNumbersHint') }}
                    </p>
                  </div>
                  <Switch
                    :checked="settings.calendar.showWeekNumbers"
                    :model-value="settings.calendar.showWeekNumbers"
                    :aria-label="$t('settings.calendar.showWeekNumbers')"
                    @update:checked="onShowWeekNumbersSwitch"
                    @update:model-value="onShowWeekNumbersSwitch"
                  />
                </div>

                <Separator />

                <section class="space-y-4">
                  <div class="space-y-1">
                    <h4 class="text-base font-bold text-foreground">
                      {{ $t('settings.calendar.data') }}
                    </h4>
                    <p class="text-muted-foreground text-sm">
                      {{ $t('settings.calendar.dataIntro') }}
                    </p>
                  </div>

                  <div class="flex items-center justify-between gap-4">
                    <div class="space-y-0.5">
                      <Label class="text-foreground text-sm font-medium">
                        {{ $t('settings.calendar.export') }}
                      </Label>
                      <p class="text-muted-foreground text-xs">
                        {{ $t('settings.calendar.exportHint') }}
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      class="h-9 shrink-0 gap-2"
                      :disabled="isExporting"
                      @click="onExportSchedules"
                    >
                      <Icon
                        v-if="isExporting"
                        icon="lucide:loader-2"
                        class="h-4 w-4 animate-spin"
                      />
                      <Icon v-else icon="lucide:upload" class="h-4 w-4" />
                      {{ $t('settings.calendar.export') }}
                    </Button>
                  </div>

                  <div class="space-y-2">
                    <div class="flex items-center justify-between gap-4">
                      <div class="space-y-0.5">
                        <Label
                          for="settings-import-mode"
                          class="text-foreground text-sm font-medium"
                        >
                          {{ $t('settings.calendar.import') }}
                        </Label>
                        <p class="text-muted-foreground text-xs">
                          {{ importModeHint }}
                        </p>
                      </div>
                      <div class="flex shrink-0 items-center gap-2">
                        <Select :model-value="importMode" @update:model-value="onImportModeChange">
                          <SelectTrigger id="settings-import-mode" class="h-9 w-40">
                            <SelectValue :placeholder="$t('settings.calendar.importMode')" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem
                              v-for="option in importModeOptions"
                              :key="option.value"
                              :value="option.value"
                            >
                              {{ option.label }}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                        <Button
                          type="button"
                          variant="outline"
                          class="h-9 gap-2"
                          :disabled="isImporting"
                          @click="onImportSchedules"
                        >
                          <Icon
                            v-if="isImporting"
                            icon="lucide:loader-2"
                            class="h-4 w-4 animate-spin"
                          />
                          <Icon v-else icon="lucide:download" class="h-4 w-4" />
                          {{ $t('settings.calendar.import') }}
                        </Button>
                      </div>
                    </div>
                  </div>

                  <p class="text-muted-foreground text-xs">
                    {{ $t('settings.calendar.dataHint') }}
                  </p>
                </section>
              </div>

              <!-- 알림 -->
              <div v-if="activeTab === 'notifications'" class="space-y-6">
                <p class="text-muted-foreground text-sm">
                  {{ $t('settings.notifications.intro') }}
                </p>

                <div class="flex items-center justify-between border-b border-neutral-100 py-2">
                  <div class="min-w-0 space-y-0.5">
                    <Label class="text-sm font-semibold">
                      {{ $t('settings.notifications.enabled') }}
                    </Label>
                    <p class="text-muted-foreground wrap-break-words text-xs">
                      {{ $t('settings.notifications.enabledHint') }}
                    </p>
                  </div>
                  <Switch
                    :checked="settings.notifications.enabled"
                    :model-value="settings.notifications.enabled"
                    :aria-label="$t('settings.notifications.enabled')"
                    @update:checked="onAppAlertSwitch"
                    @update:model-value="onAppAlertSwitch"
                  />
                </div>

                <div class="space-y-2">
                  <Label
                    for="settings-reminder-minutes"
                    class="text-foreground text-sm font-medium"
                    >{{ $t('settings.notifications.defaultTime') }}</Label
                  >
                  <Select
                    :model-value="String(settings.notifications.defaultReminderMinutes)"
                    @update:model-value="onReminderMinutesChange"
                  >
                    <SelectTrigger id="settings-reminder-minutes" class="w-full">
                      <SelectValue
                        :placeholder="$t('settings.notifications.defaultTimePlaceholder')"
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem
                        v-for="option in reminderMinuteOptions"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <p class="text-muted-foreground text-xs">
                    {{ $t('settings.notifications.defaultTimeHint') }}
                  </p>
                </div>
              </div>

              <!-- 플러그인 관리 -->
              <div v-if="activeTab === 'extensions'" class="space-y-6">
                <p class="text-muted-foreground text-sm">{{ $t('settings.extensions.intro') }}</p>

                <!-- 설치 폼 -->
                <div
                  class="rounded-xl border border-neutral-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900"
                >
                  <h4
                    class="mb-3 text-sm font-bold text-foreground dark:text-neutral-100 flex items-center gap-2"
                  >
                    <Icon icon="lucide:github" class="h-4 w-4" />
                    {{ $t('settings.extensions.installFromGithub') }}
                  </h4>
                  <div class="flex items-center gap-3">
                    <input
                      v-model="installUrl"
                      type="text"
                      placeholder="https://github.com/username/repo"
                      class="flex-1 rounded-lg border border-neutral-300 bg-transparent px-3 py-2 text-sm placeholder:text-neutral-400 focus:border-[#A68A64] focus:outline-none focus:ring-1 focus:ring-[#A68A64] dark:border-neutral-700"
                      @keydown.enter="onInstallPlugin"
                    />
                    <Button
                      type="button"
                      :disabled="!installUrl || isInstalling"
                      class="h-9 gap-2 border-none bg-[#A68A64] text-white hover:bg-[#8E7554]"
                      @click="onInstallPlugin"
                    >
                      <Icon
                        v-if="isInstalling"
                        icon="lucide:loader-2"
                        class="h-4 w-4 animate-spin"
                      />
                      <Icon v-else icon="lucide:download" class="h-4 w-4" />
                      {{ $t('settings.extensions.install') }}
                    </Button>
                  </div>

                  <Separator class="my-5" />

                  <h4
                    class="mb-3 text-sm font-bold text-foreground dark:text-neutral-100 flex items-center gap-2"
                  >
                    <Icon icon="lucide:download" class="h-4 w-4" />
                    {{ $t('settings.extensions.installFromZip') }}
                  </h4>
                  <div class="flex items-center gap-3">
                    <Button
                      type="button"
                      :disabled="isInstalling"
                      class="h-9 gap-2 border-none bg-neutral-200 text-foreground hover:bg-neutral-300 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700"
                      @click="onLocalZipSelect"
                    >
                      <Icon
                        v-if="isInstalling"
                        icon="lucide:loader-2"
                        class="h-4 w-4 animate-spin"
                      />
                      <Icon v-else icon="lucide:download" class="h-4 w-4" />
                      {{ $t('settings.extensions.selectAndInstall') }}
                    </Button>
                  </div>
                </div>

                <!-- 확장 목록 -->
                <div class="space-y-4">
                  <h4 class="text-base font-bold text-foreground dark:text-neutral-100">
                    {{ $t('settings.extensions.installed') }}
                  </h4>

                  <div
                    v-if="installedPlugins.length === 0"
                    class="flex flex-col items-center justify-center rounded-xl border border-dashed border-neutral-300 bg-neutral-50 py-12 text-center dark:border-neutral-800 dark:bg-neutral-950"
                  >
                    <div class="mb-4 rounded-full bg-neutral-200/50 p-3 dark:bg-neutral-800/50">
                      <Icon icon="lucide:puzzle" class="h-6 w-6 text-neutral-500" />
                    </div>
                    <p class="text-sm font-medium text-foreground dark:text-neutral-100">
                      {{ $t('settings.extensions.emptyTitle') }}
                    </p>
                    <p class="mt-1 text-xs text-neutral-500">
                      {{ $t('settings.extensions.emptyHint') }}
                    </p>
                  </div>

                  <div v-else class="grid gap-4 sm:grid-cols-2">
                    <div
                      v-for="plugin in installedPlugins"
                      :key="plugin.id"
                      class="flex flex-col rounded-xl border border-neutral-200 bg-white shadow-sm transition-colors hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"
                    >
                      <div class="flex items-start justify-between p-4">
                        <div class="min-w-0 flex-1">
                          <div class="flex items-center gap-2">
                            <h5
                              class="truncate font-semibold text-foreground dark:text-neutral-100"
                              :title="plugin.name"
                            >
                              {{ plugin.name }}
                            </h5>
                            <span
                              class="rounded bg-neutral-100 px-1.5 py-0.5 text-2xs font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
                              >v{{ plugin.version }}</span
                            >
                          </div>
                          <p class="mt-1 text-xs text-neutral-500">by {{ plugin.author }}</p>
                          <p
                            class="mt-2 line-clamp-2 text-sm text-neutral-600 dark:text-neutral-300"
                            :title="plugin.description"
                          >
                            {{ plugin.description }}
                          </p>
                        </div>
                      </div>

                      <div
                        class="mt-auto flex items-center justify-between border-t border-neutral-100 bg-neutral-50/50 px-4 py-3 dark:border-neutral-800 dark:bg-neutral-950/50"
                      >
                        <div class="flex items-center gap-2">
                          <Switch
                            :model-value="plugin.enabled"
                            :aria-label="$t('settings.extensions.enableAria')"
                            @update:model-value="
                              (v) => {
                                plugin.enabled = v;
                                onTogglePlugin(plugin);
                              }
                            "
                          />
                          <span
                            class="text-xs font-medium"
                            :class="plugin.enabled ? 'text-[#A68A64]' : 'text-neutral-500'"
                          >
                            {{
                              plugin.enabled
                                ? $t('settings.extensions.enabled')
                                : $t('settings.extensions.disabled')
                            }}
                          </span>
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          class="h-8 w-8 text-neutral-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/50"
                          @click="onUninstallExtension(plugin)"
                        >
                          <Icon icon="lucide:trash-2" class="h-4 w-4" />
                          <span class="sr-only">{{ $t('common.delete') }}</span>
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Extension: custom render -->
              <SettingsExtensionPanel
                v-else-if="activeExtensionTab?.render"
                :render-fn="activeExtensionTab.render"
                :panel-key="activeTab"
              />

              <!-- Extension: schema sections -->
              <div v-else-if="activeExtensionTab?.sections?.length" class="space-y-8">
                <p v-if="activeExtensionTab.extensionName" class="text-muted-foreground text-sm">
                  {{
                    $t('settings.extensions.configHeading', {
                      name: activeExtensionTab.extensionName,
                    })
                  }}
                </p>
                <ConfigSchemaForm
                  v-for="section in activeExtensionTab.sections"
                  :key="section.id"
                  v-model:values="activeExtensionDraft"
                  :items="section.items"
                  :section-title="section.title"
                  :section-description="section.description"
                />
              </div>
            </div>
          </div>

          <div
            class="bg-muted/10 mt-auto flex shrink-0 justify-end gap-3 border-t px-6 py-4 md:px-8"
          >
            <Button
              type="button"
              variant="outline"
              class="h-9 border-neutral-200 px-6 font-semibold"
              @click="handleCancel"
            >
              {{ $t('common.cancel') }}
            </Button>
            <Button
              type="button"
              class="bg-croffle-primary hover:bg-croffle-hover h-9 text-white"
              @click="handleSave"
            >
              {{ $t('common.save') }}
            </Button>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
