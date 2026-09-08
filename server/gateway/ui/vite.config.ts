import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	test: {
		// Only the framework-free modules under src/lib are covered; component
		// tests would need a browser environment this project does not set up.
		include: ['src/**/*.test.ts'],
		environment: 'node'
	},
	server: {
		proxy: {
			'/v1': 'http://localhost:8889',
			'/.well-known': 'http://localhost:8889',
			'/ui': 'http://localhost:8889'
		}
	}
});
