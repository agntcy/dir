<script lang="ts">
	import type { AICardFilterCriteria, CatalogEntry, CatalogTag } from '$lib/types';
	import {
		buildAICardFilterQuery,
		CATALOG_PAGE_SIZE,
		fetchAICardsPage,
		fetchCatalogTags,
		pageTokenForPage,
		type AICardsPage
	} from '$lib/api';
	import AICard from '$lib/components/AICard.svelte';
	import FilterSidebar from '$lib/components/FilterSidebar.svelte';
	import DetailModal from '$lib/components/DetailModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { headerStatsState } from '$lib/header-stats.svelte';
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	let aicards = $state<CatalogEntry[]>([]);
	let catalogTags = $state<CatalogTag[]>([]);
	let tagsLoading = $state(true);
	let loading = $state(true);
	let error = $state('');
	let selectedAicard = $state<CatalogEntry | null>(null);
	let catalogTotalCount = $state<number | null>(null);
	let filteredCount = $state<number | null>(null);

	let loadRequestId = 0;
	let searchDebounce: ReturnType<typeof setTimeout> | undefined;
	let listAbort: AbortController | undefined;

	let totalPages = $derived(Math.max(1, Math.ceil((filteredCount ?? 0) / CATALOG_PAGE_SIZE)));
	let resultsLabel = $derived(
		filteredCount === null || filteredCount === 0
			? ''
			: filteredCount === 1
				? '1 result'
				: `${filteredCount} results`
	);

	function parseCriteriaFromUrl(url: URL): AICardFilterCriteria {
		const params = url.searchParams;
		return {
			searchQuery: params.get('q') || '',
			mediaTypes: params.has('media') ? new Set(params.getAll('media')) : new Set(['all']),
			statusFilters: new Set(params.getAll('status')),
			activeTags: new Set(params.getAll('tag')),
			scanSafe: params.get('safe') === 'true'
		};
	}

	function updateUrl(criteria: AICardFilterCriteria, pageNum: number, selectedId: string | null) {
		const url = new URL(window.location.href);
		const p = url.searchParams;
		
		if (criteria.searchQuery) p.set('q', criteria.searchQuery);
		else p.delete('q');

		p.delete('media');
		if (criteria.mediaTypes.size > 0 && !criteria.mediaTypes.has('all')) {
			for (const m of criteria.mediaTypes) p.append('media', m);
		}

		p.delete('status');
		for (const s of criteria.statusFilters) p.append('status', s);

		p.delete('tag');
		for (const t of criteria.activeTags) p.append('tag', t);

		if (criteria.scanSafe) p.set('safe', 'true');
		else p.delete('safe');

		if (pageNum > 1) p.set('page', pageNum.toString());
		else p.delete('page');

		if (selectedId) p.set('entry', selectedId);
		else p.delete('entry');

		goto(url.pathname + url.search, { keepFocus: true, noScroll: true });
	}

	let latestCriteria = $derived(parseCriteriaFromUrl($page.url));
	let currentPage = $derived(Number($page.url.searchParams.get('page')) || 1);
	let selectedAicardId = $derived($page.url.searchParams.get('entry'));

	async function loadAICards(criteria: AICardFilterCriteria, page = 1) {
		const requestId = ++loadRequestId;
		listAbort?.abort();
		listAbort = new AbortController();
		const signal = listAbort.signal;

		loading = true;
		error = '';

		try {
			const filter = buildAICardFilterQuery(criteria);
			const result = await fetchAICardsPage({
				filter: filter || undefined,
				pageSize: CATALOG_PAGE_SIZE,
				pageToken: pageTokenForPage(page),
				signal
			});

			if (requestId !== loadRequestId) return;

			aicards = result.results;
			filteredCount = result.totalCount;
		} catch (e) {
			if (signal.aborted || requestId !== loadRequestId) return;
			error = e instanceof Error ? e.message : 'Unknown error';
			aicards = [];
			filteredCount = 0;
		} finally {
			if (requestId === loadRequestId) {
				loading = false;
			}
		}
	}

	let lastFetchKey = '';

	$effect(() => {
		const criteria = latestCriteria;
		const pageNum = currentPage;
		const filterQuery = buildAICardFilterQuery(criteria);
		const fetchKey = `${filterQuery}|${pageNum}`;
		
		if (fetchKey !== lastFetchKey) {
			lastFetchKey = fetchKey;
			untrack(() => {
				loadAICards(criteria, pageNum);
			});
		}
	});

	$effect(() => {
		if (selectedAicardId) {
			if (selectedAicard?.identifier !== selectedAicardId) {
				const found = aicards.find((c: CatalogEntry) => c.identifier === selectedAicardId);
				if (found) {
					untrack(() => { selectedAicard = found; });
				} else if (!loading) {
					fetchAICardsPage({ filter: `identifier="${selectedAicardId}"`, pageSize: 1 })
						.then((res: AICardsPage) => {
							if (res.results.length > 0 && selectedAicardId === res.results[0].identifier) {
								selectedAicard = res.results[0];
							}
						})
						.catch(() => {});
				}
			}
		} else {
			untrack(() => { selectedAicard = null; });
		}
	});

	function handleCriteriaChange(criteria: AICardFilterCriteria) {
		clearTimeout(searchDebounce);

		const delay = criteria.searchQuery.trim() ? 300 : 0;
		searchDebounce = setTimeout(() => {
			updateUrl(criteria, 1, selectedAicardId);
		}, delay);
	}

	function handlePage(page: number) {
		if (!latestCriteria || page === currentPage) return;
		updateUrl(latestCriteria, page, selectedAicardId);
		document.getElementById('ai-cards-grid')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	$effect(() => {
		if (catalogTotalCount !== null) {
			headerStatsState.set({ totalCount: catalogTotalCount });
		}
	});

	async function loadCatalogTotalCount(signal: AbortSignal) {
		try {
			const result = await fetchAICardsPage({ pageSize: 1, signal });
			if (signal.aborted) return;
			catalogTotalCount = result.totalCount;
		} catch (e) {
			if (signal.aborted) return;
			catalogTotalCount = null;
		}
	}

	async function loadCatalogTags(signal: AbortSignal) {
		tagsLoading = true;
		try {
			catalogTags = await fetchCatalogTags(signal);
		} catch (e) {
			if (signal.aborted) return;
			catalogTags = [];
		} finally {
			if (!signal.aborted) tagsLoading = false;
		}
	}

	onMount(() => {
		const tagsAbort = new AbortController();
		const statsAbort = new AbortController();
		loadCatalogTags(tagsAbort.signal);
		loadCatalogTotalCount(statsAbort.signal);

		return () => {
			tagsAbort.abort();
			statsAbort.abort();
			listAbort?.abort();
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
			<FilterSidebar {catalogTags} {tagsLoading} criteria={latestCriteria} onCriteriaChange={handleCriteriaChange} />
		</aside>

		<section class="flex-1 min-w-0">
			{#if resultsLabel}
				<p class="text-sm text-ink-medium mb-4">{resultsLabel}</p>
			{/if}

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
					<button onclick={() => latestCriteria && loadAICards(latestCriteria, currentPage)} class="mt-4 px-4 py-2 bg-brand-500 text-white text-sm font-medium rounded hover:bg-brand-600 transition">Retry</button>
				</div>
			{:else if aicards.length === 0}
				<div class="text-center py-20">
					<svg class="mx-auto h-12 w-12 text-ink-weak" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
					<p class="mt-3 text-ink-medium">No records match your filters.</p>
				</div>
			{:else}
				<div id="ai-cards-grid" class="grid gap-4 sm:grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
					{#each aicards as aicard (aicard.identifier)}
						<AICard {aicard} onclick={() => { updateUrl(latestCriteria, currentPage, aicard.identifier); }} />
					{/each}
				</div>

				<Pagination {currentPage} {totalPages} onpage={handlePage} />
			{/if}
		</section>
	</div>
</main>

{#if selectedAicard}
	<DetailModal aicard={selectedAicard} onclose={() => { updateUrl(latestCriteria, currentPage, null); }} />
{/if}
