import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import Page from './+page.svelte';
import { writable } from 'svelte/store';
import * as navigation from '$app/navigation';
import { STATUS_FILTERS } from '$lib/types';
import type { StatusFilter, StatusFilterOption } from '$lib/types';

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

function statusFilter(value: StatusFilter): StatusFilterOption {
	const option = STATUS_FILTERS.find((candidate) => candidate.value === value);
	if (!option) throw new Error(`${value} is not a status filter`);
	return option;
}

const trusted = statusFilter('trusted');
const identityVerified = statusFilter('identityVerified');

/** The single-record request the page sends to learn whether a hidden status filter has anything to match. */
function probeUrl(clause: string): string {
	return `/v1/agents?page_size=1&filter=${encodeURIComponent(clause)}`;
}

const PROBE_URLS = new Set(
	STATUS_FILTERS.filter((opt) => opt.hiddenUntilMatch).map((opt) => probeUrl(opt.clause))
);

function jsonResponse(body: unknown) {
	return Promise.resolve({
		ok: true,
		json: () => Promise.resolve(body)
	});
}

/** Answers every request the page makes; `matchingRecords` is what each hidden-filter probe sees. */
function mockFetch(matchingRecords = 0) {
	const fetchMock = vi.fn().mockImplementation((url) => {
		const target = url.toString();
		if (target.includes('/v1/tags')) {
			return jsonResponse({ tags: [] });
		}
		if (PROBE_URLS.has(target)) {
			return jsonResponse({ results: [], totalCount: matchingRecords });
		}
		if (target.includes('/v1/agents')) {
			return jsonResponse({
				results: [
					{ identifier: 'entry1', displayName: 'Test Entry 1', type: 'application/a2a-agent-card+json' }
				],
				totalCount: 1
			});
		}
		return Promise.reject(new Error('not mocked: ' + url));
	});
	vi.stubGlobal('fetch', fetchMock);
	return fetchMock;
}

beforeEach(() => {
	vi.clearAllMocks();
	mockPage.set({ url: new URL('http://localhost/') });
	window.HTMLElement.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

describe('Catalog URL Syncing', () => {
	let fetchMock: ReturnType<typeof mockFetch>;

	beforeEach(() => {
		fetchMock = mockFetch();
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
			expect(fetchMock).toHaveBeenCalledWith(
				expect.stringContaining('/v1/agents?page_size=18&filter=displayName%3Dfoo+AND+type%3Dapplication%2Fa2a-agent-card%2Bjson&page_token=18'),
				expect.any(Object)
			);
		});
	});

	it('updates URL when filter is changed', async () => {
		render(Page);
		
		// Initially fetch is called without filter
		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith(
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

	it('filters by identityVerified when the URL carries that status', async () => {
		mockPage.set({ url: new URL(`http://localhost/?status=${identityVerified.value}`) });

		render(Page);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith(
				`/v1/agents?page_size=18&filter=${encodeURIComponent(identityVerified.clause)}`,
				expect.any(Object)
			);
		});
	});

	it('drops status values the sidebar no longer offers', async () => {
		mockPage.set({ url: new URL('http://localhost/?status=verified') });

		render(Page);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith('/v1/agents?page_size=18', expect.any(Object));
		});
	});
});

describe('Hidden status filters', () => {
	it('stay hidden while no record would match them', async () => {
		const fetchMock = mockFetch(0);

		render(Page);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith(probeUrl(identityVerified.clause), expect.any(Object));
		});
		await waitFor(() => expect(screen.getByLabelText(trusted.label)).toBeTruthy());

		expect(screen.queryByLabelText(identityVerified.label)).toBeNull();
	});

	it('appear once the probe finds a matching record', async () => {
		mockFetch(1);

		render(Page);

		await waitFor(() => expect(screen.getByLabelText(identityVerified.label)).toBeTruthy());
	});

	it('stay visible while active, even with nothing left to match', async () => {
		mockPage.set({ url: new URL(`http://localhost/?status=${identityVerified.value}`) });
		mockFetch(0);

		render(Page);

		await waitFor(() => {
			const checkbox = screen.getByLabelText(identityVerified.label) as HTMLInputElement;
			expect(checkbox.checked).toBe(true);
		});
	});
});
