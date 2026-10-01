/** Why an extension install failed. Main tags its errors with `[code]`; the renderer localizes it. */
export const EXTENSION_INSTALL_ERROR_CODES = [
  'invalid-source',
  'repo-or-ref-not-found',
  'network',
  'invalid-archive',
  'manifest-missing',
  'manifest-invalid',
  'incompatible-engine',
] as const;

export type ExtensionInstallErrorCode = (typeof EXTENSION_INSTALL_ERROR_CODES)[number];

/** The `[code]` an install error message carries (IPC keeps only the message), if any. */
export function extensionInstallErrorCode(message: string): ExtensionInstallErrorCode | null {
  const code = message.match(/\[([a-z-]+)\]/)?.[1];
  return EXTENSION_INSTALL_ERROR_CODES.find((known) => known === code) ?? null;
}

/** A GitHub repository (and optional branch / tag / commit) to install an extension from. */
export type GitHubSource = {
  owner: string;
  repo: string;
  ref?: string;
};

export type ParsedGitHubSource = GitHubSource | { error: 'invalid-source' };

// GitHub owner names: alphanumerics and single hyphens. Repo names also allow `.` and `_`.
const OWNER = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$/;
const REPO = /^[A-Za-z0-9._-]{1,100}$/;

const invalid = { error: 'invalid-source' } as const;

/**
 * Accepts `owner/repo`, `https://github.com/owner/repo` (optional `www.`, trailing `/` or `.git`),
 * `…/tree/<ref>` and a `@<ref>` or `#<ref>` suffix on any of them.
 */
export function parseGitHubSource(input: string): ParsedGitHubSource {
  let rest = input.trim();
  if (!rest) {
    return invalid;
  }

  let ref: string | undefined;
  const suffix = rest.match(/^([^@#]+)[@#](.+)$/);
  if (suffix) {
    rest = suffix[1];
    ref = suffix[2].trim();
  }

  const url = rest.match(/^(?:https?:\/\/)?(?:www\.)?github\.com\/(.+)$/i);
  if (url) {
    rest = url[1];
  } else if (/^[a-z][a-z0-9+.-]*:/i.test(rest)) {
    // A URL on another host.
    return invalid;
  }

  const parts = rest.replace(/\/+$/, '').split('/');
  const [owner, rawRepo, ...tail] = parts;
  const repo = rawRepo?.replace(/\.git$/i, '');
  if (!owner || !repo || !OWNER.test(owner) || !REPO.test(repo) || repo === '.' || repo === '..') {
    return invalid;
  }

  if (tail.length > 0) {
    // Only `/tree/<ref>` (refs may contain slashes) is meaningful after owner/repo.
    if (tail[0] !== 'tree' || tail.length < 2 || ref !== undefined || !url) {
      return invalid;
    }
    ref = tail.slice(1).join('/');
  }

  if (ref !== undefined && (!ref || /\s|\.\.|^[/-]/.test(ref))) {
    return invalid;
  }

  return ref ? { owner, repo, ref } : { owner, repo };
}
