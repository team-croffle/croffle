/**
 * Module hooks for `node --test`: the shared code in src/common is written for the bundler
 * (relative imports without `.ts`, JSON imports without attributes, `rrule` through its ESM
 * build). These hooks let a test import it as it is. Registered by test/support/register.ts.
 */
const RRULE_ESM = '/node_modules/rrule/dist/esm/';

export async function resolve(specifier, context, next) {
  if (specifier === 'rrule') {
    return next('rrule/dist/esm/index.js', context);
  }
  try {
    return await next(specifier, context);
  } catch (error) {
    const relative = specifier.startsWith('./') || specifier.startsWith('../');
    if (!relative || !['ERR_MODULE_NOT_FOUND', 'ERR_UNSUPPORTED_DIR_IMPORT'].includes(error.code)) {
      throw error;
    }
    const fromRrule = context.parentURL?.includes(RRULE_ESM);
    const candidates = fromRrule
      ? [`${specifier}.js`, `${specifier}/index.js`]
      : [`${specifier}.ts`, `${specifier}/index.ts`];
    for (const candidate of candidates) {
      try {
        return await next(candidate, context);
      } catch {
        // try the next candidate
      }
    }
    throw error;
  }
}

export async function load(url, context, next) {
  if (url.endsWith('.json') && !context.importAttributes?.type) {
    return next(url, {
      ...context,
      importAttributes: { ...context.importAttributes, type: 'json' },
    });
  }
  if (url.includes(RRULE_ESM)) {
    // The package has no "type": "module", but dist/esm is ES module code.
    return next(url, { ...context, format: 'module' });
  }
  return next(url, context);
}
