import fs from 'node:fs';
import path from 'node:path';

import type { CroffleManifest } from '@croffledev/common';
import { parseGitHubSource } from '@croffledev/common';
import { app, net, protocol } from 'electron';
import JSZip from 'jszip';

import type { ExtensionInfoRow } from '../database/schema';
import { logger } from '../logger';
import { downloadGitHubSource } from './github-download';
import { extensionInfoService } from './info-service';
import { ExtensionInstallError } from './install-error';
import {
  getExtensionDir,
  isInsideDir,
  MANIFEST_FILENAME,
  satisfiesCroffleEngine,
} from './manifest';
import { clearItem as clearSession } from './session-service';
import { clear as clearStorage } from './storage';

class ExtensionManager {
  private extensionDir = getExtensionDir();

  constructor() {
    app.whenReady().then(() => {
      this.registerProtocol();
    });
  }

  private registerProtocol() {
    protocol.handle('extension', async (req) => {
      let url = req.url.replace('extension://', '');

      const queryIndex = url.indexOf('?');
      if (queryIndex !== -1) {
        url = url.substring(0, queryIndex);
      }

      const safePath = path.normalize(url).replace(/^(\.\.(\/|\\|$))+/, '');
      const localPath = path.join(this.extensionDir, safePath);
      if (!isInsideDir(this.extensionDir, localPath)) {
        return new Response(null, { status: 404 });
      }

      const response = await net.fetch(`file://${localPath}`);

      const headers = new Headers(response.headers);
      headers.set('Access-Control-Allow-Origin', '*');
      headers.set('Access-Control-Allow-Methods', 'GET');
      headers.set('Cross-Origin-Resource-Policy', 'cross-origin');

      return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers,
      });
    });
  }

  private readAndValidateManifest(dir: string): CroffleManifest {
    const manifestPath = path.join(dir, MANIFEST_FILENAME);
    if (!fs.existsSync(manifestPath)) {
      throw new ExtensionInstallError(
        'manifest-missing',
        `${MANIFEST_FILENAME} not found at the archive root`,
      );
    }

    let manifest: CroffleManifest;
    try {
      manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8')) as CroffleManifest;
    } catch (err) {
      throw new ExtensionInstallError(
        'manifest-invalid',
        `${MANIFEST_FILENAME} is not valid JSON: ${String(err)}`,
      );
    }
    if (!manifest.id || !manifest.name || !manifest.version || !manifest.author) {
      throw new ExtensionInstallError(
        'manifest-invalid',
        `${MANIFEST_FILENAME} is missing required fields (id, name, version, author)`,
      );
    }

    const appVersion = app.getVersion();
    if (!satisfiesCroffleEngine(appVersion, manifest.engines?.croffle)) {
      throw new ExtensionInstallError(
        'incompatible-engine',
        `Extension requires Croffle ${manifest.engines?.croffle}, but app is ${appVersion}`,
      );
    }

    return manifest;
  }

  private async extractZip(
    buffer: Buffer | ArrayBuffer,
  ): Promise<{ tempDir: string; contentDir: string }> {
    let zip: JSZip;
    try {
      zip = await JSZip.loadAsync(buffer);
    } catch (err) {
      throw new ExtensionInstallError('invalid-archive', `Not a zip archive: ${String(err)}`);
    }
    const tempDir = path.join(app.getPath('temp'), `croffle-extension-${Date.now()}`);
    await fs.promises.mkdir(tempDir, { recursive: true });

    // Zip-slip: an entry like `../x` or an absolute path must not write outside tempDir.
    const unsafeEntry = Object.keys(zip.files).find(
      (relativePath) => !isInsideDir(tempDir, path.join(tempDir, relativePath)),
    );
    if (unsafeEntry) {
      await fs.promises.rm(tempDir, { recursive: true, force: true });
      throw new ExtensionInstallError(
        'invalid-archive',
        `Archive entry escapes the install folder: ${unsafeEntry}`,
      );
    }

    const writes: Promise<void>[] = [];

    zip.forEach((relativePath, zipEntry) => {
      const targetPath = path.join(tempDir, relativePath);

      if (zipEntry.dir) {
        fs.mkdirSync(targetPath, { recursive: true });
      } else {
        const dirname = path.dirname(targetPath);
        if (!fs.existsSync(dirname)) {
          fs.mkdirSync(dirname, { recursive: true });
        }

        writes.push(
          zipEntry
            .async('nodebuffer')
            .then((content) => fs.promises.writeFile(targetPath, content)),
        );
      }
    });

    await Promise.all(writes);

    const dirs = fs.readdirSync(tempDir);
    const nestedRoot =
      dirs.length === 1 && fs.statSync(path.join(tempDir, dirs[0])).isDirectory() ? dirs[0] : '';
    const contentDir = nestedRoot ? path.join(tempDir, nestedRoot) : tempDir;

    return { tempDir, contentDir };
  }

  private async finalizeInstall(contentDir: string, tempDir: string): Promise<ExtensionInfoRow> {
    try {
      const manifest = this.readAndValidateManifest(contentDir);
      const finalDir = path.join(this.extensionDir, manifest.id);

      if (fs.existsSync(finalDir)) {
        fs.rmSync(finalDir, { recursive: true, force: true });
      }

      fs.mkdirSync(this.extensionDir, { recursive: true });
      fs.renameSync(contentDir, finalDir);

      return await extensionInfoService.installExtension({
        id: manifest.id,
        name: manifest.name,
        version: manifest.version,
        author: manifest.author,
        description: manifest.description,
        main: manifest.main,
        enabled: true,
      });
    } finally {
      if (fs.existsSync(tempDir)) {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    }
  }

  async installFromLocalZip(zipPath: string) {
    const buffer = fs.readFileSync(zipPath);
    const { tempDir, contentDir } = await this.extractZip(buffer);
    return this.finalizeInstall(contentDir, tempDir);
  }

  /** `input`: `owner/repo`, a github.com URL, `…/tree/<ref>`, or any of them with `@<ref>`. */
  async installFromGitHub(input: string) {
    const source = parseGitHubSource(input);
    if ('error' in source) {
      throw new ExtensionInstallError('invalid-source', `Not a GitHub repository: ${input}`);
    }

    const buffer = await downloadGitHubSource(source);
    const { tempDir, contentDir } = await this.extractZip(buffer);
    return this.finalizeInstall(contentDir, tempDir);
  }

  /**
   * Remove extension files, storage, session, then DB row.
   * Best-effort purge of side state before deleting the info record.
   */
  async uninstallExtension(extensionId: string): Promise<boolean> {
    try {
      await clearStorage(extensionId);
    } catch (err) {
      logger.error('ExtensionManager', `Failed to clear storage for ${extensionId}:`, err);
    }

    try {
      clearSession(extensionId);
    } catch (err) {
      logger.error('ExtensionManager', `Failed to clear session for ${extensionId}:`, err);
    }

    const finalDir = path.join(this.extensionDir, extensionId);
    try {
      if (fs.existsSync(finalDir)) {
        fs.rmSync(finalDir, { recursive: true, force: true });
      }
    } catch (err) {
      logger.error('ExtensionManager', `Failed to remove files for ${extensionId}:`, err);
    }

    return extensionInfoService.uninstallExtension(extensionId);
  }
}

export const extensionManager = new ExtensionManager();
