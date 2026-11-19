/** Import first in a test that loads src/common code written for the bundler. */
import { register } from 'node:module';

register('./ts-resolve.mjs', import.meta.url);
