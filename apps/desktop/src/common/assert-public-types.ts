/**
 * Ensures desktop-owned public DTOs stay aligned with `@croffledev/croffle-types`.
 * Host-only types (FeatureView, etc.) are intentionally not asserted here.
 */
import type {
  ConfigItemSchema as PublicConfigItemSchema,
  ConfigurationSectionContribution as PublicConfigurationSectionContribution,
  ConfigurationTabManifest as PublicConfigurationTabManifest,
  ContextMenuManifest as PublicContextMenuManifest,
  CroffleManifest as PublicCroffleManifest,
  ExtensionContributes as PublicExtensionContributes,
  ExtensionInfo as PublicExtensionInfo,
  Schedule as PublicSchedule,
  ScheduleException as PublicScheduleException,
  SearchQuery as PublicSearchQuery,
  Tag as PublicTag,
  ViewManifest as PublicViewManifest,
} from '@croffledev/croffle-types';

import type {
  ConfigItemSchema,
  ConfigurationSectionContribution,
  ConfigurationTabManifest,
  ContextMenuManifest,
  CroffleManifest,
  ExtensionContributes,
  ExtensionInfo,
  Schedule,
  ScheduleException,
  SearchQuery,
  Tag,
  ViewManifest,
} from './models';
import type { AssertEqual, AssertTrue } from './type-utils';

/** Referenced so unused-type lint does not strip the asserts. */
export type PublicTypeIntegrity = {
  tag: AssertTrue<AssertEqual<Tag, PublicTag>>;
  schedule: AssertTrue<AssertEqual<Schedule, PublicSchedule>>;
  scheduleException: AssertTrue<AssertEqual<ScheduleException, PublicScheduleException>>;
  searchQuery: AssertTrue<AssertEqual<SearchQuery, PublicSearchQuery>>;
  configItemSchema: AssertTrue<AssertEqual<ConfigItemSchema, PublicConfigItemSchema>>;
  configurationSectionContribution: AssertTrue<
    AssertEqual<ConfigurationSectionContribution, PublicConfigurationSectionContribution>
  >;
  configurationTabManifest: AssertTrue<
    AssertEqual<ConfigurationTabManifest, PublicConfigurationTabManifest>
  >;
  viewManifest: AssertTrue<AssertEqual<ViewManifest, PublicViewManifest>>;
  contextMenuManifest: AssertTrue<AssertEqual<ContextMenuManifest, PublicContextMenuManifest>>;
  extensionContributes: AssertTrue<AssertEqual<ExtensionContributes, PublicExtensionContributes>>;
  croffleManifest: AssertTrue<AssertEqual<CroffleManifest, PublicCroffleManifest>>;
  extensionInfo: AssertTrue<AssertEqual<ExtensionInfo, PublicExtensionInfo>>;
};
