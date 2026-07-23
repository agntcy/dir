<script lang="ts">
	import type { AICardFilterCriteria, CatalogEntry } from '$lib/types';
	import { buildAICardFilterQuery, CATALOG_HYDRATION_PAGE_SIZE, CATALOG_PAGE_SIZE, fetchAICardsPage } from '$lib/api';
	import { applyClientFilters, collectSortedTags, hasActiveClientFilters, mergeSortedTags } from '$lib/utils';
	import AICard from '$lib/components/AICard.svelte';
	import FilterSidebar from '$lib/components/FilterSidebar.svelte';
	import DetailModal from '$lib/components/DetailModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { headerStatsState } from '$lib/header-stats.svelte';
	import { onMount } from 'svelte';

	let aicards = $state<CatalogEntry[]>([]);
	let filteredAicards = $state<CatalogEntry[]>([]);
	let catalogTags = $state<string[]>([]);
	let loading = $state(true);
	let catalogHydrating = $state(false);
	let hydrationError = $state('');
	let error = $state('');
	let currentPage = $state(1);
	let selectedAicard = $state<CatalogEntry | null>(null);
	let totalCount = $state(0);

	let latestCriteria = $state<AICardFilterCriteria | null>(null);
	let loadedServerFilter = $state<string | null>(null);
	let loadRequestId = 0;
	let searchDebounce: ReturnType<typeof setTimeout> | undefined;
	let backgroundAbort: AbortController | undefined;

	let totalPages = $derived(
		Math.max(1, Math.ceil(filteredAicards.length / CATALOG_PAGE_SIZE))
	);
	let pageItems = $derived(
		filteredAicards.slice(
			(currentPage - 1) * CATALOG_PAGE_SIZE,
			currentPage * CATALOG_PAGE_SIZE
		)
	);

	function applyFilters(
		loaded: CatalogEntry[],
		criteria: AICardFilterCriteria,
		resetPage = true
	) {
		filteredAicards = applyClientFilters(loaded, criteria);
		if (resetPage) currentPage = 1;
	}

	function appendAICardPage(newAicards: CatalogEntry[], criteria: AICardFilterCriteria) {
		if (newAicards.length === 0) return;

		aicards = aicards.concat(newAicards);
		catalogTags = mergeSortedTags(catalogTags, newAicards);
		if (hasActiveClientFilters(criteria)) {
			filteredAicards = filteredAicards.concat(applyClientFilters(newAicards, criteria));
		} else {
			filteredAicards = aicards;
		}
	}

	function hydrationMatchesServerFilter(filter: string): boolean {
		return latestCriteria !== null && buildAICardFilterQuery(latestCriteria) === filter;
	}

	async function hydrateCatalog(
		filter: string,
		pageToken: string,
		requestId: number,
		signal: AbortSignal
	) {
		let nextPageToken = pageToken;

		while (nextPageToken) {
			if (signal.aborted || requestId !== loadRequestId) return;
			if (!hydrationMatchesServerFilter(filter)) return;

			const page = await fetchAICardsPage({
				filter: filter || undefined,
				pageSize: CATALOG_HYDRATION_PAGE_SIZE,
				pageToken: nextPageToken,
				signal
			});

			if (signal.aborted || requestId !== loadRequestId) return;
			if (!hydrationMatchesServerFilter(filter)) return;

			appendAICardPage(page.results, latestCriteria!);
			nextPageToken = page.nextPageToken;
		}
	}

	async function loadAICards(criteria: AICardFilterCriteria) {
		const requestId = ++loadRequestId;
		backgroundAbort?.abort();
		backgroundAbort = new AbortController();
		const signal = backgroundAbort.signal;

		const filter = buildAICardFilterQuery(criteria);
		loading = true;
		catalogHydrating = false;
		hydrationError = '';
		error = '';

		try {
			const firstPage = await fetchAICardsPage({
				filter: filter || undefined,
				signal
			});

			if (requestId !== loadRequestId) return;

			aicards = firstPage.results;
			catalogTags = collectSortedTags(firstPage.results);
			totalCount = firstPage.totalCount;
			loadedServerFilter = filter;
			applyFilters(aicards, criteria);
			loading = false;

			if (firstPage.nextPageToken) {
				catalogHydrating = true;
				try {
					await hydrateCatalog(filter, firstPage.nextPageToken, requestId, signal);
				} catch (e) {
					if (signal.aborted || requestId !== loadRequestId) return;
					hydrationError =
						e instanceof Error ? e.message : 'Failed to load remaining records';
				}
			}
		} catch (e) {
			if (signal.aborted || requestId !== loadRequestId) return;
			error = e instanceof Error ? e.message : 'Unknown error';
			aicards = [];
			filteredAicards = [];
			catalogTags = [];
			totalCount = 0;
		} finally {
			if (requestId === loadRequestId) {
				loading = false;
				catalogHydrating = false;
			}
		}
	}

	function handleCriteriaChange(criteria: AICardFilterCriteria) {
		latestCriteria = criteria;
		const serverFilter = buildAICardFilterQuery(criteria);

		if (serverFilter !== loadedServerFilter) {
			clearTimeout(searchDebounce);
			backgroundAbort?.abort();
			const delay = criteria.searchQuery.trim() ? 300 : 0;
			searchDebounce = setTimeout(() => {
				if (latestCriteria) loadAICards(latestCriteria);
			}, delay);
			return;
		}

		applyFilters(aicards, criteria);
	}

	function handlePage(page: number) {
		currentPage = page;
		document.getElementById('ai-cards-grid')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	$effect(() => {
		headerStatsState.set({
			totalCount,
			catalogHydrating,
			hydrationError
		});
	});

	onMount(() => {
		const initial: AICardFilterCriteria = {
			searchQuery: '',
			mediaTypes: new Set(['all']),
			statusFilters: new Set(),
			activeTags: new Set(),
			scanSafe: false
		};
		latestCriteria = initial;
		loadAICards(initial);

		return () => {
			backgroundAbort?.abort();
			clearTimeout(searchDebounce);
			headerStatsState.set(null);
		};
	});
</script>

<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="mb-6">
		<h2 class="font-display text-2xl font-bold text-ink-strong leading-tight">Explore</h2>
		<p class="mt-1 text-sm text-ink-medium max-w-3xl">
			Browse the secure directory of verified public records published to the Agent Directory Service.
		</p>
	</div>

	<div class="flex flex-col lg:flex-row gap-6">
		<aside class="lg:w-64 flex-shrink-0">
			<FilterSidebar allTags={catalogTags} {catalogHydrating} onchange={handleCriteriaChange} />
		</aside>

		<section class="flex-1 min-w-0">
			{#if loading}
				<div class="flex items-center justify-center py-20">
					<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-500"></div>
					<span class="ml-3 text-ink-medium">Loading records...</span>
				</div>
			{:else if error}
				<div class="text-center py-20">
					<svg class="mx-auto h-12 w-12 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/>
					</svg>
					<p class="mt-3 text-red-600 font-medium">Failed to load records: {error}</p>
					<button onclick={() => latestCriteria && loadAICards(latestCriteria)} class="mt-4 px-4 py-2 bg-brand-500 text-white text-sm font-medium rounded hover:bg-brand-600 transition">Retry</button>
				</div>
			{:else if pageItems.length === 0}
				<div class="text-center py-20">
					<svg class="mx-auto h-12 w-12 text-ink-weak" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
					<p class="mt-3 text-ink-medium">No records match your filters.</p>
				</div>
			{:else}
				<div id="ai-cards-grid" class="grid gap-4 sm:grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
					{#each pageItems as aicard (aicard.identifier)}
						<AICard {aicard} onclick={() => { selectedAicard = aicard; }} />
					{/each}
				</div>

				<Pagination {currentPage} {totalPages} onpage={handlePage} />
				{#if catalogHydrating}
					<p class="mt-3 text-center text-xs text-ink-weak">Loading more records for filters and pagination…</p>
				{:else if hydrationError}
					<p class="mt-3 text-center text-xs text-amber-700">
						Some records could not be loaded. Filters and pagination may be incomplete.
					</p>
				{/if}
			{/if}
		</section>
	</div>
</main>

{#if selectedAicard}
	<DetailModal aicard={selectedAicard} onclose={() => { selectedAicard = null; }} />
{/if}
