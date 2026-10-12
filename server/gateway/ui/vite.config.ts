import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'jsdom'
	},
	resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
	server: {
		proxy: {
			'/v1': 'http://localhost:8889',
			'/.well-known': 'http://localhost:8889',
			'/ui': 'http://localhost:8889'
		}
	}
});
