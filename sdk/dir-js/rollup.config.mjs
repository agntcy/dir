import json from '@rollup/plugin-json';
import {readFileSync} from 'fs';
import typescript from 'rollup-plugin-typescript2';

const pkg = JSON.parse(
  readFileSync(new URL('./package.json', import.meta.url), 'utf8'),
);

const rollupPlugins = [
  typescript({
    tsconfigOverride: {
      exclude: ['test/**'],
    },
  }),
  json({
    preferConst: true,
  }),
];

const externalDeps = [
  'node:os',
  'node:path',
  'node:process',
  'node:fs',
  'node:child_process',
  '@buf/agntcy_dir.bufbuild_es'
];

export default [
  // Cross ES module (dist/index.mjs)
  {
    input: 'src/index.ts',
    output: {
      file: pkg.exports['.']['import'],
      format: 'es',
      sourcemap: true,
    },
    plugins: rollupPlugins,
    external: externalDeps,
  },

  // Cross CJS module (dist/index.cjs)
  {
    input: 'src/index.ts',
    output: {
      file: pkg.exports['.']['require'],
      format: 'cjs',
      sourcemap: true,
    },
    plugins: rollupPlugins,
    external: externalDeps,
  },
];
