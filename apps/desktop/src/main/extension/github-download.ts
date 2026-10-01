import type { GitHubSource } from '@croffledev/common';

import { ExtensionInstallError } from './install-error';

type Fetch = typeof fetch;

// Branches tried in order when the GitHub API cannot tell the default branch (e.g. rate limit).
const FALLBACK_BRANCHES = ['main', 'master'];

async function request(fetchFn: Fetch, url: string, init?: RequestInit): Promise<Response> {
  try {
    return await fetchFn(url, init);
  } catch (err) {
    throw new ExtensionInstallError('network', `Could not reach ${url}: ${String(err)}`);
  }
}

/**
 * The repository's default branch, `null` when the API cannot answer (rate limit, outage).
 * A missing (or private) repository is reported right away.
 */
async function fetchDefaultBranch(fetchFn: Fetch, source: GitHubSource): Promise<string | null> {
  const url = `https://api.github.com/repos/${source.owner}/${source.repo}`;
  const resp = await request(fetchFn, url, {
    headers: { Accept: 'application/vnd.github+json' },
  });
  if (resp.status === 404) {
    throw new ExtensionInstallError(
      'repo-or-ref-not-found',
      `Repository ${source.owner}/${source.repo} not found`,
    );
  }
  if (!resp.ok) {
    return null;
  }
  const body = (await resp.json()) as { default_branch?: unknown };
  return typeof body.default_branch === 'string' ? body.default_branch : null;
}

async function downloadRef(fetchFn: Fetch, source: GitHubSource, ref: string) {
  // codeload serves branches, tags and commits alike.
  const url = `https://codeload.github.com/${source.owner}/${source.repo}/zip/${encodeURI(ref)}`;
  const resp = await request(fetchFn, url);
  if (resp.status === 404) {
    return null;
  }
  if (!resp.ok) {
    throw new ExtensionInstallError('network', `Download failed (${resp.status}): ${url}`);
  }
  return resp.arrayBuffer();
}

/** Download the repository snapshot zip for `source.ref`, or for the default branch. */
export async function downloadGitHubSource(
  source: GitHubSource,
  fetchFn: Fetch = fetch,
): Promise<ArrayBuffer> {
  const name = `${source.owner}/${source.repo}`;
  const refs = source.ref
    ? [source.ref]
    : await fetchDefaultBranch(fetchFn, source).then((branch) =>
        branch ? [branch] : FALLBACK_BRANCHES,
      );

  for (const ref of refs) {
    const zip = await downloadRef(fetchFn, source, ref);
    if (zip) {
      return zip;
    }
  }

  throw new ExtensionInstallError(
    'repo-or-ref-not-found',
    source.ref ? `${name}@${source.ref} not found` : `${name} not found`,
  );
}
