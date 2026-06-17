<script lang="ts">
	import type { AgentFilterCriteria, CatalogEntry } from '$lib/types';
	import { buildAgentFilterQuery, CATALOG_HYDRATION_PAGE_SIZE, CATALOG_PAGE_SIZE, fetchAgentsPage } from '$lib/api';
	import { applyClientFilters, collectSortedTags, hasActiveClientFilters, mergeSortedTags } from '$lib/utils';
	import AgentCard from '$lib/components/AgentCard.svelte';
	import FilterSidebar from '$lib/components/FilterSidebar.svelte';
	import DetailModal from '$lib/components/DetailModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import DisclaimerBanner from '$lib/components/DisclaimerBanner.svelte';
	import { headerStatsState } from '$lib/header-stats.svelte';
	import { onMount } from 'svelte';

	let agents = $state<CatalogEntry[]>([]);
	let filteredAgents = $state<CatalogEntry[]>([]);
	let catalogTags = $state<string[]>([]);
	let loading = $state(true);
	let catalogHydrating = $state(false);
	let hydrationError = $state('');
	let error = $state('');
	let currentPage = $state(1);
	let selectedAgent = $state<CatalogEntry | null>(null);

	let latestCriteria = $state<AgentFilterCriteria | null>(null);
	let loadedServerFilter = $state<string | null>(null);
	let loadRequestId = 0;
	let searchDebounce: ReturnType<typeof setTimeout> | undefined;
	let backgroundAbort: AbortController | undefined;

	let totalPages = $derived(
		Math.max(1, Math.ceil(filteredAgents.length / CATALOG_PAGE_SIZE))
	);
	let pageItems = $derived(
		filteredAgents.slice(
			(currentPage - 1) * CATALOG_PAGE_SIZE,
			currentPage * CATALOG_PAGE_SIZE
		)
	);

	function applyFilters(
		loaded: CatalogEntry[],
		criteria: AgentFilterCriteria,
		resetPage = true
	) {
		filteredAgents = applyClientFilters(loaded, criteria);
		if (resetPage) currentPage = 1;
	}

	function appendAgentPage(newAgents: CatalogEntry[], criteria: AgentFilterCriteria) {
		if (newAgents.length === 0) return;

		agents = agents.concat(newAgents);
		catalogTags = mergeSortedTags(catalogTags, newAgents);
		if (hasActiveClientFilters(criteria)) {
			filteredAgents = filteredAgents.concat(applyClientFilters(newAgents, criteria));
		} else {
			filteredAgents = agents;
		}
	}

	function hydrationMatchesServerFilter(filter: string): boolean {
		return latestCriteria !== null && buildAgentFilterQuery(latestCriteria) === filter;
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

			const page = await fetchAgentsPage({
				filter: filter || undefined,
				pageSize: CATALOG_HYDRATION_PAGE_SIZE,
				pageToken: nextPageToken,
				signal
			});

			if (signal.aborted || requestId !== loadRequestId) return;
			if (!hydrationMatchesServerFilter(filter)) return;

			appendAgentPage(page.results, latestCriteria!);
			nextPageToken = page.nextPageToken;
		}
	}

	async function loadAgents(criteria: AgentFilterCriteria) {
		const requestId = ++loadRequestId;
		backgroundAbort?.abort();
		backgroundAbort = new AbortController();
		const signal = backgroundAbort.signal;

		const filter = buildAgentFilterQuery(criteria);
		loading = true;
		catalogHydrating = false;
		hydrationError = '';
		error = '';

		try {
			const firstPage = await fetchAgentsPage({
				filter: filter || undefined,
				signal
			});

			if (requestId !== loadRequestId) return;

			agents = firstPage.results;
			catalogTags = collectSortedTags(firstPage.results);
			loadedServerFilter = filter;
			applyFilters(agents, criteria);
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
			agents = [];
			filteredAgents = [];
			catalogTags = [];
		} finally {
			if (requestId === loadRequestId) {
				loading = false;
				catalogHydrating = false;
			}
		}
	}

	function handleCriteriaChange(criteria: AgentFilterCriteria) {
		latestCriteria = criteria;
		const serverFilter = buildAgentFilterQuery(criteria);

		if (serverFilter !== loadedServerFilter) {
			clearTimeout(searchDebounce);
			backgroundAbort?.abort();
			const delay = criteria.searchQuery.trim() ? 300 : 0;
			searchDebounce = setTimeout(() => {
				if (latestCriteria) loadAgents(latestCriteria);
			}, delay);
			return;
		}

		applyFilters(agents, criteria);
	}

	function handlePage(page: number) {
		currentPage = page;
		document.getElementById('agents-grid')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	$effect(() => {
		headerStatsState.set({
			count: filteredAgents.length,
			catalogHydrating,
			hydrationError
		});
		return () => {
			headerStatsState.set(null);
		};
	});

	onMount(() => {
		const initial: AgentFilterCriteria = {
			searchQuery: '',
			mediaTypes: new Set(['all']),
			statusFilter: 'all',
			activeTags: new Set()
		};
		latestCriteria = initial;
		loadAgents(initial);

		return () => {
			backgroundAbort?.abort();
			clearTimeout(searchDebounce);
		};
	});
</script>

<DisclaimerBanner />

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
					<button onclick={() => latestCriteria && loadAgents(latestCriteria)} class="mt-4 px-4 py-2 bg-brand-500 text-white text-sm font-medium rounded hover:bg-brand-600 transition">Retry</button>
				</div>
			{:else if pageItems.length === 0}
				<div class="text-center py-20">
					<svg class="mx-auto h-12 w-12 text-ink-weak" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
					<p class="mt-3 text-ink-medium">No records match your filters.</p>
				</div>
			{:else}
				<div id="agents-grid" class="grid gap-4 sm:grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
					{#each pageItems as agent (agent.identifier)}
						<AgentCard {agent} onclick={() => { selectedAgent = agent; }} />
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

{#if selectedAgent}
	<DetailModal agent={selectedAgent} onclose={() => { selectedAgent = null; }} />
{/if}
