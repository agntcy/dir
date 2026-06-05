import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: '../static',
			assets: '../static',
			fallback: 'index.html'
		}),
		paths: {
			base: ''
		},
		version: {
			name: '0'
		}
	}
};

export default config;
