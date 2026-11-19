<script setup lang="ts">
  import {
    AppCloseBehavior,
    AppSettingLanguage,
    AppSettingTheme,
    CalendarTimeFormat,
    CalendarView,
    AppEventType,
    applySetting,
    CalendarWeekStartDay,
    extensionInstallErrorCode,
    PendingPatches,
  } from '@croffledev/common';
  import type { ConfigurationTabContribution, ExtensionInfo } from '@croffledev/common';
  import type { AppSettings } from '@croffledev/croffle-types';
  import { ref, onBeforeUnmount, onMounted, watch, computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { toast } from 'vue-sonner';

  import ConfigSchemaForm from '@/components/settings/config-schema-form.vue';
  import SettingsExtensionPanel from '@/components/settings/settings-extension-panel.vue';
  import SyncPanel from '@/components/settings/sync-panel.vue';
  import { Badge } from '@/components/ui/badge';
  import { Button } from '@/components/ui/button';
  import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
  } from '@/components/ui/dialog';
  import { Icon } from '@/components/ui/icon';
  import { Input } from '@/components/ui/input';
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
  import UpdateCheck from '@/components/update-check.vue';
  import { extensionLoader } from '@/services/extension-loader';
  import { croffleInternal } from '@/services/host-bridge';
  import { useAppSettingsStore } from '@/stores/app-settings-store';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useSettingsStore } from '@/stores/settings-store';
  import { useUiStore } from '@/stores/ui-store';
  import {
    DEFAULT_ACCENT_HUE,
    DEFAULT_ACCENT_PRESET_HEX,
    hexToHue,
    hueToPreviewHex,
    huesMatch,
  } from '@/utils/appearance';
  import { mergeWithSchemaDefaults } from '@/utils/extension-configuration-schema';

  import pkg from '../../../../package.json';

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
  const uiStore = useUiStore();
  const { t } = useI18n();
  const appVersion = pkg.version;

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
    { id: 'sync', label: t('settings.tabs.sync'), icon: 'lucide:refresh-cw' },
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
    const stored = await croffleInternal.extensions.configuration.get<Record<string, unknown>>(
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
      const stored = await croffleInternal.extensions.configuration.get<Record<string, unknown>>(
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
      patches = new PendingPatches(loaded);
      showPatches();
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

  /** A localized reason when main tagged the error with an install code, else the fallback. */
  const installErrorMessage = (err: unknown, fallbackKey: string) => {
    const message = err instanceof Error ? err.message : String(err);
    const code = extensionInstallErrorCode(message);
    if (code) {
      const key = code.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase());
      return t(`settings.errors.installReasons.${key}`);
    }
    return t(fallbackKey, { error: message });
  };

  const onInstallPlugin = async () => {
    if (!installUrl.value) {
      return;
    }
    isInstalling.value = true;
    try {
      const plugin = await croffleInternal.extensions.info.install({
        id: installUrl.value,
      });
      installUrl.value = '';
      await fetchInstalledPlugins();
      await extensionLoader.loadPluginById(plugin.id);
    } catch (err) {
      toast.error(installErrorMessage(err, 'settings.errors.install'));
    } finally {
      isInstalling.value = false;
    }
  };

  const onLocalZipSelect = async () => {
    isInstalling.value = true;
    try {
      const result = await croffleInternal.extensions.info.installFromLocal();
      if (result) {
        await fetchInstalledPlugins();
        await extensionLoader.loadPluginById(result.id);
      }
    } catch (err) {
      toast.error(installErrorMessage(err, 'settings.errors.installLocal'));
    } finally {
      isInstalling.value = false;
    }
  };

  const onTogglePlugin = async (plugin: ExtensionInfo) => {
    try {
      await croffleInternal.extensions.info.toggle(plugin.id, plugin.enabled);
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
    const confirmed = await uiStore.openConfirm({
      title: t('settings.extensions.uninstallTitle'),
      description: t('settings.extensions.uninstallConfirm', { name: plugin.name }),
      confirmLabel: t('settings.extensions.uninstall'),
      confirmVariant: 'destructive',
    });
    if (!confirmed) {
      return;
    }
    try {
      await extensionLoader.unloadPlugin(plugin.id);
      await croffleInternal.extensions.info.uninstall(plugin.id);
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
        // Closing does not drop a change still waiting for its save.
        flushPending();
        return;
      }
      if (originalSettings.value) {
        void refreshFromStore();
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

  const reapplyPersistedAppearance = (value: AppSettings) => {
    appSettingsStore.setThemeDraft(value.general.theme);
    appSettingsStore.setAccentHueDraft(value.appearance?.accentHue ?? DEFAULT_ACCENT_HUE);
  };

  const sameJson = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

  /**
   * App settings apply as soon as a control changes: only that field is sent, and main merges it
   * into what it has stored. The screen shows the latest stored settings with the patches still
   * saving on top (see PendingPatches), so a change saved elsewhere is never taken for an edit,
   * and a failed patch shows the latest stored value again.
   */
  let patches: PendingPatches<AppSettings> | null = null;

  const showPatches = () => {
    if (patches) {
      originalSettings.value = cloneSettings(patches.stored);
      settings.value = patches.view();
    }
  };

  const receiveStored = (latest: AppSettings) => {
    if (!patches) {
      return;
    }
    patches.receive(latest);
    showPatches();
    // SETTINGS_UPDATE also re-applies the stored look app-wide; keep the screen's unsent colour
    // and theme on it.
    if (settings.value) {
      reapplyPersistedAppearance(settings.value);
    }
  };

  const refreshFromStore = async () => {
    try {
      receiveStored(await croffle.settings.getAll());
    } catch (error) {
      toast.error(t('settings.errors.load', { error: errorMessage(error) }));
    }
  };

  /** Saves one changed field. */
  const edit = async <S extends keyof AppSettings>(
    section: S,
    field: keyof AppSettings[S] & string,
    value: AppSettings[S][keyof AppSettings[S]],
  ) => {
    if (!patches) {
      return;
    }
    const { id, patch } = patches.edit(section, field, value);
    showPatches();
    await send(id, patch);
  };

  /**
   * Sends one patch. Main stores it and broadcasts SETTINGS_UPDATE before replying, so the stored
   * value arrives through `receiveStored`; the reply only ends the patch.
   */
  const send = async (id: number, patch: Partial<AppSettings>) => {
    if (!patches) {
      return;
    }
    try {
      await croffleInternal.settings.update(patch);
      patches.settle(id);
      showPatches();
    } catch (error) {
      patches.settle(id);
      showPatches();
      toast.error(t('settings.errors.save', { error: errorMessage(error) }));
      if (settings.value) {
        reapplyPersistedAppearance(settings.value);
      }
    }
  };

  /** Applies extension configuration that changed on screen. */
  const persistExtensionDrafts = async () => {
    for (const tab of settingsStore.sortedExtensionTabs) {
      const compositeId = settingsStore.getTabCompositeId(tab);
      const draft = extensionDrafts.value[compositeId];
      const stored = originalExtensionDrafts.value[compositeId] ?? {};
      if (!draft || sameJson(draft, stored)) {
        continue;
      }
      const snapshot = JSON.parse(JSON.stringify(draft)) as Record<string, unknown>;
      const result = await applySetting(stored, snapshot, async (value) => {
        await croffleInternal.extensions.configuration.set(tab.extensionId, value);
        return value;
      });
      originalExtensionDrafts.value[compositeId] = result.value;
      if (!result.ok) {
        const reason = result.reason === 'failed' ? errorMessage(result.error) : result.message;
        toast.error(t('settings.errors.save', { error: reason }));
        // Put back only what this run tried to save; a newer edit stays and is saved next.
        if (sameJson(extensionDrafts.value[compositeId], snapshot)) {
          extensionDrafts.value[compositeId] = JSON.parse(JSON.stringify(result.value));
        }
      }
    }
  };

  // Saves run one at a time, in order; a failed one does not stop the next.
  let saving: Promise<void> = Promise.resolve();
  const enqueue = (run: () => Promise<void>) => {
    saving = saving.then(run, run);
    return saving;
  };

  // A short pause groups quick changes (dragging the colour picker) into one save.
  const PERSIST_DELAY_MS = 250;
  let accentTimer: ReturnType<typeof setTimeout> | null = null;
  /** The colour patch shown on screen but not sent yet (the picker is still moving). */
  let accentPatchId: number | null = null;

  /** The colour picker previews at once and sends the hue after a short pause. */
  const sendAccent = async () => {
    if (!patches || accentPatchId === null) {
      return;
    }
    const id = accentPatchId;
    accentPatchId = null;
    const hue = patches.view().appearance?.accentHue ?? DEFAULT_ACCENT_HUE;
    await send(id, patches.revise(id, hue));
  };
  let extensionTimer: ReturnType<typeof setTimeout> | null = null;

  const flushPending = () => {
    if (accentTimer) {
      clearTimeout(accentTimer);
      accentTimer = null;
      void sendAccent();
    }
    if (extensionTimer) {
      clearTimeout(extensionTimer);
      extensionTimer = null;
      void enqueue(persistExtensionDrafts);
    }
  };

  watch(
    extensionDrafts,
    () => {
      if (extensionTimer) {
        clearTimeout(extensionTimer);
      }
      extensionTimer = setTimeout(() => {
        extensionTimer = null;
        void enqueue(persistExtensionDrafts);
      }, PERSIST_DELAY_MS);
    },
    { deep: true },
  );

  // Settings saved elsewhere (title bar theme, another window) reach the screen at once.
  let offSettingsUpdate: (() => void) | null = null;
  onMounted(() => {
    offSettingsUpdate = croffle.event.on(AppEventType.SETTINGS_UPDATE, (payload) => {
      receiveStored(payload as AppSettings);
    });
  });
  onBeforeUnmount(() => {
    offSettingsUpdate?.();
    flushPending();
  });

  /** A select changed one field (the value is the option's string). */
  const onSelect = (section: 'general' | 'calendar', field: string, value: unknown) => {
    if (typeof value !== 'string') {
      return;
    }
    void edit(section, field as never, value as never);
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
    void edit('general', key, asBool(v));
  };

  const onThemeChange = (theme: unknown) => {
    if (!settings.value || typeof theme !== 'string') {
      return;
    }
    appSettingsStore.setThemeDraft(theme as AppSettingTheme);
    void edit('general', 'theme', theme as AppSettingTheme);
  };

  const onAccentHexChange = (hex: string) => {
    if (!patches) {
      return;
    }
    // First preset is the exact Croffle primary hue (avoid round-trip drift).
    const hue =
      hex.toUpperCase() === DEFAULT_ACCENT_PRESET_HEX.toUpperCase()
        ? DEFAULT_ACCENT_HUE
        : hexToHue(hex);
    if (accentPatchId !== null && patches.has(accentPatchId)) {
      patches.revise(accentPatchId, hue);
    } else {
      accentPatchId = patches.edit('appearance', 'accentHue', hue).id;
    }
    showPatches();
    appSettingsStore.setAccentHueDraft(hue);
    if (accentTimer) {
      clearTimeout(accentTimer);
    }
    accentTimer = setTimeout(() => {
      accentTimer = null;
      void sendAccent();
    }, PERSIST_DELAY_MS);
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
    if (v === null) {
      return;
    }
    void edit('notifications', 'defaultReminderMinutes', Number(v));
  };

  const onAppAlertSwitch = (v: unknown) => {
    void edit('notifications', 'enabled', asBool(v));
  };

  const onShowWeekNumbersSwitch = (v: unknown) => {
    void edit('calendar', 'showWeekNumbers', asBool(v));
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
    if (typeof value !== 'string') {
      return;
    }
    void edit('general', 'closeBehavior', value as AppCloseBehavior);
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
      const result = await croffleInternal.calendar.schedules.exportSchedulesToFile();
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
      const result = await croffleInternal.calendar.schedules.importScheduleFromFile(
        importMode.value,
      );
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
      class="h-[90vh] max-h-235 w-[96vw] gap-0 overflow-hidden border-none p-0 sm:w-[94vw] sm:max-w-none lg:w-[92vw]"
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
            <Button
              v-for="tab in allTabs"
              :key="tab.id"
              type="button"
              variant="ghost"
              :class="[
                'w-full justify-start',
                activeTab === tab.id
                  ? 'bg-croffle-sidebar-selected text-croffle-primary-strong hover:bg-croffle-sidebar-selected hover:text-croffle-primary-strong'
                  : 'text-muted-foreground',
              ]"
              @click="
                'extensionTab' in tab && tab.extensionTab
                  ? onExtensionTabClick(tab.extensionTab)
                  : (activeTab = tab.id)
              "
            >
              <Icon v-if="typeof tab.icon === 'string'" :icon="tab.icon" />
              <component v-else :is="tab.icon" />
              {{ tab.label }}
            </Button>
          </nav>
        </div>

        <!-- 우측 콘텐츠 -->
        <div class="flex min-h-0 min-w-0 flex-1 flex-col">
          <div class="shrink-0 px-6 py-6">
            <h3 class="text-2xl font-bold wrap-break-word">
              {{ activeTabLabel }}
            </h3>
          </div>

          <div class="flex-1 overflow-y-auto px-6 pb-8">
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

                  <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0 space-y-0.5">
                      <Label for="settings-language" class="text-foreground text-sm font-medium">
                        {{ $t('settings.general.language') }}
                      </Label>
                    </div>
                    <Select
                      :model-value="settings.general.language"
                      @update:model-value="(v: unknown) => onSelect('general', 'language', v)"
                    >
                      <SelectTrigger id="settings-language" class="w-60 shrink-0">
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

                  <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0 space-y-0.5">
                      <Label for="settings-theme" class="text-foreground text-sm font-medium">
                        {{ $t('settings.general.theme') }}
                      </Label>
                    </div>
                    <Select
                      :model-value="settings.general.theme"
                      @update:model-value="onThemeChange"
                    >
                      <SelectTrigger id="settings-theme" class="w-60 shrink-0">
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
                        class="hover:bg-muted/50 relative flex size-8 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed"
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
                  <div class="flex items-start justify-between gap-4">
                    <div class="space-y-0.5">
                      <Label class="text-foreground text-sm font-medium">
                        {{ $t('settings.general.updateCheck') }}
                      </Label>
                      <p class="text-muted-foreground text-xs">
                        {{ $t('settings.general.updateCheckHint', { version: appVersion }) }}
                      </p>
                    </div>
                    <UpdateCheck class="w-60 shrink-0" />
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

                  <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0 space-y-0.5">
                      <Label
                        for="settings-close-behavior"
                        class="text-foreground text-sm font-medium"
                      >
                        {{ $t('settings.general.closeBehavior') }}
                      </Label>
                      <p class="text-muted-foreground text-xs">
                        {{ $t('settings.general.closeBehaviorHint') }}
                      </p>
                    </div>
                    <Select
                      :model-value="settings.general.closeBehavior"
                      @update:model-value="onCloseBehaviorChange"
                    >
                      <SelectTrigger id="settings-close-behavior" class="w-60 shrink-0">
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
                  </div>
                </section>
              </div>

              <!-- 캘린더 -->
              <div v-if="activeTab === 'calendar'" class="space-y-6">
                <p class="text-muted-foreground text-sm">{{ $t('settings.calendar.intro') }}</p>

                <div class="flex items-center justify-between gap-4">
                  <div class="min-w-0 space-y-0.5">
                    <Label for="settings-default-view" class="text-foreground text-sm font-medium">
                      {{ $t('settings.calendar.defaultView') }}
                    </Label>
                  </div>
                  <Select
                    :model-value="settings.calendar.defaultView"
                    @update:model-value="(v: unknown) => onSelect('calendar', 'defaultView', v)"
                  >
                    <SelectTrigger id="settings-default-view" class="w-60 shrink-0">
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

                <div class="flex items-center justify-between gap-4">
                  <div class="min-w-0 space-y-0.5">
                    <Label for="settings-week-start" class="text-foreground text-sm font-medium">
                      {{ $t('settings.calendar.weekStart') }}
                    </Label>
                  </div>
                  <Select
                    :model-value="settings.calendar.weekStartDay"
                    @update:model-value="(v: unknown) => onSelect('calendar', 'weekStartDay', v)"
                  >
                    <SelectTrigger id="settings-week-start" class="w-60 shrink-0">
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

                <div class="flex items-center justify-between gap-4">
                  <div class="min-w-0 space-y-0.5">
                    <Label for="settings-time-format" class="text-foreground text-sm font-medium">
                      {{ $t('settings.calendar.timeFormat') }}
                    </Label>
                  </div>
                  <Select
                    :model-value="settings.calendar.timeFormat"
                    @update:model-value="(v: unknown) => onSelect('calendar', 'timeFormat', v)"
                  >
                    <SelectTrigger id="settings-time-format" class="w-60 shrink-0">
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
                      class="shrink-0"
                      :disabled="isExporting"
                      @click="onExportSchedules"
                    >
                      <Icon v-if="isExporting" icon="lucide:loader-2" class="animate-spin" />
                      <Icon v-else icon="lucide:upload" />
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
                          <SelectTrigger id="settings-import-mode" class="w-60 shrink-0">
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
                          :disabled="isImporting"
                          @click="onImportSchedules"
                        >
                          <Icon v-if="isImporting" icon="lucide:loader-2" class="animate-spin" />
                          <Icon v-else icon="lucide:download" />
                          {{ $t('settings.calendar.import') }}
                        </Button>
                      </div>
                    </div>
                  </div>
                </section>
              </div>

              <!-- 알림 -->
              <div v-if="activeTab === 'notifications'" class="space-y-6">
                <p class="text-muted-foreground text-sm">
                  {{ $t('settings.notifications.intro') }}
                </p>

                <div class="flex items-center justify-between border-b py-2">
                  <div class="min-w-0 space-y-0.5">
                    <Label class="text-foreground text-sm font-medium">
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

                <div class="flex items-center justify-between gap-4">
                  <div class="min-w-0 space-y-0.5">
                    <Label
                      for="settings-reminder-minutes"
                      class="text-foreground text-sm font-medium"
                      >{{ $t('settings.notifications.defaultTime') }}</Label
                    >
                    <p class="text-muted-foreground text-xs">
                      {{ $t('settings.notifications.defaultTimeHint') }}
                    </p>
                  </div>
                  <Select
                    :model-value="String(settings.notifications.defaultReminderMinutes)"
                    @update:model-value="onReminderMinutesChange"
                  >
                    <SelectTrigger id="settings-reminder-minutes" class="w-60 shrink-0">
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
                </div>
              </div>

              <SyncPanel v-if="activeTab === 'sync'" />

              <!-- 플러그인 관리 -->
              <div v-if="activeTab === 'extensions'" class="space-y-6">
                <p class="text-muted-foreground text-sm">{{ $t('settings.extensions.intro') }}</p>

                <!-- 설치 폼 -->
                <div class="bg-card rounded-xl border p-5 shadow-sm">
                  <h4 class="mb-3 text-sm font-bold text-foreground flex items-center gap-2">
                    <Icon icon="lucide:github" class="size-4" />
                    {{ $t('settings.extensions.installFromGithub') }}
                  </h4>
                  <div class="flex items-center gap-3">
                    <Input
                      v-model="installUrl"
                      type="text"
                      :placeholder="$t('settings.extensions.githubPlaceholder')"
                      class="flex-1"
                      @keydown.enter="onInstallPlugin"
                    />
                    <Button
                      type="button"
                      :disabled="!installUrl || isInstalling"
                      @click="onInstallPlugin"
                    >
                      <Icon v-if="isInstalling" icon="lucide:loader-2" class="animate-spin" />
                      <Icon v-else icon="lucide:download" />
                      {{ $t('settings.extensions.install') }}
                    </Button>
                  </div>
                  <p class="mt-2 text-xs text-muted-foreground">
                    {{ $t('settings.extensions.githubHint') }}
                  </p>

                  <Separator class="my-5" />

                  <h4 class="mb-3 text-sm font-bold text-foreground flex items-center gap-2">
                    <Icon icon="lucide:download" class="size-4" />
                    {{ $t('settings.extensions.installFromZip') }}
                  </h4>
                  <div class="flex items-center gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      :disabled="isInstalling"
                      @click="onLocalZipSelect"
                    >
                      <Icon v-if="isInstalling" icon="lucide:loader-2" class="animate-spin" />
                      <Icon v-else icon="lucide:download" />
                      {{ $t('settings.extensions.selectAndInstall') }}
                    </Button>
                  </div>
                </div>

                <!-- 확장 목록 -->
                <div class="space-y-4">
                  <h4 class="text-base font-bold text-foreground">
                    {{ $t('settings.extensions.installed') }}
                  </h4>

                  <div
                    v-if="installedPlugins.length === 0"
                    class="bg-muted/30 flex flex-col items-center justify-center rounded-xl border border-dashed py-12 text-center"
                  >
                    <div class="mb-4 rounded-full bg-muted p-3">
                      <Icon icon="lucide:puzzle" class="size-6 text-muted-foreground" />
                    </div>
                    <p class="text-sm font-medium text-foreground">
                      {{ $t('settings.extensions.emptyTitle') }}
                    </p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      {{ $t('settings.extensions.emptyHint') }}
                    </p>
                  </div>

                  <div v-else class="grid gap-4 sm:grid-cols-2">
                    <div
                      v-for="plugin in installedPlugins"
                      :key="plugin.id"
                      class="bg-card flex flex-col rounded-xl border shadow-sm"
                    >
                      <div class="flex items-start justify-between p-4">
                        <div class="min-w-0 flex-1">
                          <div class="flex items-center gap-2">
                            <h5 class="truncate font-semibold text-foreground" :title="plugin.name">
                              {{ plugin.name }}
                            </h5>
                            <Badge variant="secondary">v{{ plugin.version }}</Badge>
                          </div>
                          <p class="mt-1 text-xs text-muted-foreground">by {{ plugin.author }}</p>
                          <p
                            class="mt-2 line-clamp-2 text-sm text-muted-foreground"
                            :title="plugin.description"
                          >
                            {{ plugin.description }}
                          </p>
                        </div>
                      </div>

                      <div
                        class="bg-muted/20 mt-auto flex items-center justify-between border-t px-4 py-3"
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
                            :class="
                              plugin.enabled
                                ? 'text-croffle-primary-strong'
                                : 'text-muted-foreground'
                            "
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
                          size="icon-sm"
                          class="text-destructive hover:bg-destructive/10 hover:text-destructive"
                          @click="onUninstallExtension(plugin)"
                        >
                          <Icon icon="lucide:trash-2" />
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
                  :key="`${activeTab}:${section.id}`"
                  v-model:values="activeExtensionDraft"
                  :items="section.items"
                  :section-title="section.title"
                  :section-description="section.description"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
