import { defineStore } from 'pinia';
import { ref, shallowRef } from 'vue';

export type ChoiceOption<T extends string = string> = {
  value: T;
  label: string;
  variant?: 'default' | 'outline' | 'destructive';
  disabled?: boolean;
};

export type ChoiceDialogOptions<T extends string = string> = {
  title: string;
  description: string;
  choices: ChoiceOption<T>[];
  /** Shown under the description, e.g. why a choice is disabled. */
  hint?: string;
  cancelLabel?: string;
};

/**
 * Multi-button dialog (the confirm dialog only has confirm / cancel).
 * `openChoice` resolves with the picked value, or `null` when dismissed.
 */
export const useChoiceStore = defineStore('choice', () => {
  const isOpen = ref(false);
  const options = shallowRef<ChoiceDialogOptions | null>(null);
  let resolver: ((value: string | null) => void) | null = null;

  const resolveChoice = (value: string | null) => {
    const resolve = resolver;
    resolver = null;
    isOpen.value = false;
    resolve?.(value);
  };

  const openChoice = <T extends string>(dialog: ChoiceDialogOptions<T>): Promise<T | null> => {
    if (resolver) {
      resolveChoice(null);
    }
    options.value = dialog;
    isOpen.value = true;
    return new Promise<T | null>((resolve) => {
      resolver = resolve as (value: string | null) => void;
    });
  };

  return { isOpen, options, openChoice, resolveChoice };
});
