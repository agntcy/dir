import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import Page from './+page.svelte';
import { readable, writable } from 'svelte/store';
import * as navigation from '$app/navigation';
import * as stores from '$app/stores';

// Mock SvelteKit stores and navigation
vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

const mockPage = writable({
	url: new URL('http://localhost/')
});

vi.mock('$app/stores', () => ({
	page: {
		subscribe: (fn: any) => mockPage.subscribe(fn)
	}
}));

describe('Catalog URL Syncing', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mockPage.set({ url: new URL('http://localhost/') });
		
		// Mock fetch globally
		global.fetch = vi.fn().mockImplementation((url) => {
			if (url.toString().includes('/v1/tags')) {
				return Promise.resolve({
					ok: true,
					json: () => Promise.resolve({ tags: [] })
				});
			}
			if (url.toString().includes('/v1/agents')) {
				return Promise.resolve({
					ok: true,
					json: () => Promise.resolve({
						results: [
							{ identifier: 'entry1', displayName: 'Test Entry 1', type: 'application/a2a-agent-card+json' }
						],
						totalCount: 1
					})
				});
			}
			return Promise.reject(new Error('not mocked: ' + url));
		});
		
		window.HTMLElement.prototype.scrollIntoView = vi.fn();
	});

	it('initializes component with search parameters from URL', async () => {
		// Set URL to have search params
		mockPage.set({ url: new URL('http://localhost/?page=2&q=foo&media=application%2Fa2a-agent-card%2Bjson') });
		
		render(Page);
		
		// The search input should be initialized with 'foo'
		await waitFor(() => {
			const searchInput = screen.getByLabelText('Search', { exact: false }) as HTMLInputElement;
			expect(searchInput.value).toBe('foo');
		});
		
		// Wait for fetch to be called and verify the query includes the filter
		await waitFor(() => {
			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('/v1/agents?page_size=18&filter=displayName%3Dfoo+AND+type%3Dapplication%2Fa2a-agent-card%2Bjson&page_token=18'),
				expect.any(Object)
			);
		});
	});

	it('updates URL when filter is changed', async () => {
		render(Page);
		
		// Initially fetch is called without filter
		await waitFor(() => {
			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('/v1/agents?page_size=18'),
				expect.any(Object)
			);
		});
		
		// Type into search
		const searchInput = screen.getByLabelText('Search', { exact: false });
		await fireEvent.input(searchInput, { target: { value: 'bar' } });
		
		// Delay for debounce
		await new Promise(r => setTimeout(r, 350));
		
		// goto should have been called
		expect(navigation.goto).toHaveBeenCalledWith(
			expect.stringContaining('?q=bar'),
			expect.objectContaining({ keepFocus: true })
		);
	});
});
