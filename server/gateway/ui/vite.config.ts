import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig(({ mode }) => ({
	plugins: [tailwindcss(), sveltekit()],
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'jsdom'
	},
	resolve: {
		conditions: mode === 'test' ? ['browser'] : [],
	},
	server: {
		proxy: {
			'/v1': 'http://localhost:8889',
			'/.well-known': 'http://localhost:8889',
			'/ui': 'http://localhost:8889'
		}
	}
}));
