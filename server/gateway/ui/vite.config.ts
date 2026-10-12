import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'jsdom'
	},
	// Vitest runs in Node and would otherwise pick Svelte's "server" export.
	// Leave resolve alone for production so the client bundle stays on the
	// browser entry. An empty conditions list overrides Vite's defaults and
	// ships SSR onDestroy into the static UI (P is null, then P.r throws).
	resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
	server: {
		proxy: {
			'/v1': 'http://localhost:8889',
			'/.well-known': 'http://localhost:8889',
			'/ui': 'http://localhost:8889'
		}
	}
});
