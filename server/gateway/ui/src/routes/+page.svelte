<script lang="ts">
	import type { AgentFilterCriteria, CatalogEntry } from '$lib/types';
	import { buildAgentFilterQuery, CATALOG_PAGE_SIZE, fetchAgentsPage } from '$lib/api';
	import { applyClientFilters } from '$lib/utils';
	import AgentCard from '$lib/components/AgentCard.svelte';
	import FilterSidebar from '$lib/components/FilterSidebar.svelte';
	import DetailModal from '$lib/components/DetailModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import DisclaimerBanner from '$lib/components/DisclaimerBanner.svelte';
	import { onMount } from 'svelte';

	let agents = $state<CatalogEntry[]>([]);
	let filteredAgents = $state<CatalogEntry[]>([]);
	let loading = $state(true);
	let catalogHydrating = $state(false);
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

	async function hydrateCatalog(
		criteria: AgentFilterCriteria,
		filter: string,
		pageToken: string,
		requestId: number,
		signal: AbortSignal
	) {
		let nextPageToken = pageToken;

		while (nextPageToken) {
			if (signal.aborted || requestId !== loadRequestId) return;

			const page = await fetchAgentsPage({
				filter: filter || undefined,
				pageToken: nextPageToken,
				signal
			});

			if (signal.aborted || requestId !== loadRequestId) return;

			agents = agents.concat(page.results);
			applyFilters(agents, criteria, false);
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
		error = '';

		try {
			const firstPage = await fetchAgentsPage({
				filter: filter || undefined,
				signal
			});

			if (requestId !== loadRequestId) return;

			agents = firstPage.results;
			loadedServerFilter = filter;
			applyFilters(agents, criteria);
			loading = false;

			if (firstPage.nextPageToken) {
				catalogHydrating = true;
				await hydrateCatalog(criteria, filter, firstPage.nextPageToken, requestId, signal);
			}
		} catch (e) {
			if (signal.aborted || requestId !== loadRequestId) return;
			error = e instanceof Error ? e.message : 'Unknown error';
			agents = [];
			filteredAgents = [];
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

<header class="bg-surface-light border-b border-line sticky top-0 z-30">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-2.5 flex items-center justify-between">
		<div class="flex items-center gap-3">
			<svg class="h-7" viewBox="0 0 492 112" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path d="M436.006 111.94H-0.00390625V45.31C-0.00390625 20.29 20.2861 0 45.3061 0H491.826L435.996 111.94H436.006ZM40.8361 39.2L24.2661 94.15H50.3261L55.5561 75.07H71.7961V94.15H96.8761V17.83H65.2561C51.2961 17.83 45.8461 22.52 40.8361 39.2ZM60.6861 56.54L64.2861 43.68C66.0261 37.36 66.7961 36.48 70.6061 36.48H71.8061V56.54H60.6861ZM144.746 96.33C153.146 96.33 161.756 94.69 168.406 91.97V48.8H144.416V75.62H143.106C134.386 75.62 128.606 68.42 128.606 56.87C128.606 43.46 136.456 36.92 149.536 36.92C157.606 36.92 164.256 39.54 168.506 44.22V19.25C162.836 16.85 154.766 15.65 147.026 15.65C119.556 15.65 101.776 31.89 101.776 57.52C101.776 80.09 116.826 96.33 144.736 96.33H144.746ZM175.056 94.15H199.586V55.45C214.956 70.6 219.756 80.42 219.756 94.15H243.196V17.83H218.666V43.56L196.536 17.83H175.056V94.15ZM268.056 94.15H293.676V38.55H296.186C304.146 38.55 310.146 40.29 313.626 44.22V17.84H248.106V44.22C251.596 40.3 257.596 38.55 265.546 38.55H268.056V94.15ZM315.816 55.99C315.816 79.87 332.496 96.33 356.486 96.33C363.356 96.33 369.676 94.91 373.166 92.73V67.76C370.436 72.01 365.646 74.19 359.646 74.19C349.066 74.19 342.636 67.21 342.636 55.98C342.636 44.75 349.066 37.77 359.646 37.77C365.646 37.77 370.436 39.95 373.166 44.2V19.23C369.676 17.05 363.356 15.63 356.486 15.63C332.496 15.63 315.816 32.09 315.816 55.97V55.99ZM387.776 94.15H415.036L452.866 17.83H425.396L414.166 43.34L403.046 17.83H375.356L400.646 69.07L387.786 94.15H387.776Z" fill="#187ADC"/>
			</svg>
			<span class="text-gray-300 text-lg font-light">|</span>
			<div>
				<h1 class="text-lg font-medium text-gray-700 leading-tight">AI Catalog</h1>
				<p class="text-xs text-gray-500 leading-tight mt-0.5">powered by Agent Directory Service</p>
			</div>
		</div>
		<div class="text-sm text-ink-medium text-right">
			<span class="font-semibold text-ink-strong">{filteredAgents.length}</span>
			{#if catalogHydrating}
				<span class="text-ink-weak">+</span>
			{/if}
			agents indexed
			{#if catalogHydrating}
				<p class="text-xs text-ink-weak mt-0.5">Loading full catalog…</p>
			{/if}
		</div>
	</div>
</header>

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
			<FilterSidebar {agents} {catalogHydrating} onchange={handleCriteriaChange} />
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
				{/if}
			{/if}
		</section>
	</div>
</main>

{#if selectedAgent}
	<DetailModal agent={selectedAgent} onclose={() => { selectedAgent = null; }} />
{/if}
