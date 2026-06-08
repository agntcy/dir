<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { extractEntryTypes, extractShortTag, hasTrustManifest } from '$lib/utils';

	interface Props {
		agents: CatalogEntry[];
		onfilter: (filtered: CatalogEntry[]) => void;
	}

	let { agents, onfilter }: Props = $props();

	let searchQuery = $state('');
	let mediaTypes = $state<Set<string>>(new Set(['all']));
	let statusFilter = $state('all');
	let activeTags = $state<Set<string>>(new Set());
	let tagSearch = $state('');

	let allTags = $derived(
		Array.from(new Set(agents.flatMap((a) => a.tags || []))).sort()
	);

	let visibleTags = $derived(
		tagSearch ? allTags.filter((t) => t.toLowerCase().includes(tagSearch.toLowerCase())) : allTags
	);

	function applyFilters() {
		const filtered = agents.filter((agent) => {
			if (searchQuery) {
				const q = searchQuery.toLowerCase();
				const name = (agent.displayName || '').toLowerCase();
				const desc = (agent.description || '').toLowerCase();
				if (!name.includes(q) && !desc.includes(q)) return false;
			}

			if (!mediaTypes.has('all')) {
				const types = extractEntryTypes(agent);
				if (!types.some((t) => mediaTypes.has(t))) return false;
			}

			if (activeTags.size > 0) {
				const agentTags = new Set(agent.tags || []);
				let hasAny = false;
				for (const t of activeTags) {
					if (agentTags.has(t)) { hasAny = true; break; }
				}
				if (!hasAny) return false;
			}

			if (statusFilter === 'trusted' || statusFilter === 'verified') {
				if (!hasTrustManifest(agent)) return false;
			}

			return true;
		});
		onfilter(filtered);
	}

	$effect(() => {
		searchQuery; mediaTypes; statusFilter; activeTags;
		applyFilters();
	});

	function handleMediaType(value: string, checked: boolean) {
		const next = new Set(mediaTypes);
		if (value === 'all' && checked) {
			next.clear();
			next.add('all');
		} else if (value !== 'all') {
			next.delete('all');
			if (checked) next.add(value);
			else next.delete(value);
			if (next.size === 0) next.add('all');
		}
		mediaTypes = next;
	}

	function handleTag(tag: string, checked: boolean) {
		const next = new Set(activeTags);
		if (checked) next.add(tag);
		else next.delete(tag);
		activeTags = next;
	}

	const MEDIA_TYPE_OPTIONS = [
		{ value: 'all', label: 'All' },
		{ value: 'application/a2a-agent-card+json', label: 'A2A Agent' },
		{ value: 'application/mcp-server+json', label: 'MCP Server' },
		{ value: 'application/ai-skill+md', label: 'SKILL' }
	];
</script>

<div class="bg-white rounded-xl border border-gray-200 p-4 space-y-5 sticky top-20 max-h-[calc(100vh-6rem)] flex flex-col overflow-hidden">
	<div class="flex-shrink-0">
		<label for="search" class="block text-sm font-medium text-gray-700 mb-1.5">Search</label>
		<input
			type="text"
			id="search"
			placeholder="Filter by name or description..."
			class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
			bind:value={searchQuery}
		/>
	</div>

	<div class="flex-shrink-0">
		<span class="block text-sm font-medium text-gray-700 mb-1.5">Media Type</span>
		<div class="space-y-1.5">
			{#each MEDIA_TYPE_OPTIONS as opt}
				<label class="flex items-center gap-2 text-sm cursor-pointer">
					<input
						type="checkbox"
						checked={mediaTypes.has(opt.value)}
						onchange={(e) => handleMediaType(opt.value, (e.target as HTMLInputElement).checked)}
						class="rounded border-gray-300 text-brand-600 focus:ring-brand-500"
					/>
					<span>{opt.label}</span>
				</label>
			{/each}
		</div>
	</div>

	<div class="flex-shrink-0">
		<span class="block text-sm font-medium text-gray-700 mb-1.5">Status</span>
		<div class="space-y-1.5">
			{#each ['all', 'trusted', 'verified'] as value}
				<label class="flex items-center gap-2 text-sm cursor-pointer">
					<input type="radio" name="status" {value} checked={statusFilter === value}
						onchange={() => { statusFilter = value; }}
						class="border-gray-300 text-brand-600 focus:ring-brand-500"
					/>
					<span class="capitalize">{value}</span>
				</label>
			{/each}
		</div>
	</div>

	<div class="flex-1 flex flex-col min-h-0">
		<span class="block text-sm font-medium text-gray-700 mb-1.5 flex-shrink-0">Tags</span>
		<input
			type="text"
			placeholder="Filter tags..."
			class="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent mb-2 flex-shrink-0"
			bind:value={tagSearch}
		/>
		<div class="space-y-1.5 overflow-y-auto flex-1">
			{#each visibleTags as tag}
				<label class="flex items-center gap-2 text-sm cursor-pointer">
					<input
						type="checkbox"
						checked={activeTags.has(tag)}
						onchange={(e) => handleTag(tag, (e.target as HTMLInputElement).checked)}
						class="rounded border-gray-300 text-brand-600 focus:ring-brand-500"
					/>
					<span class="truncate" title={tag}>{extractShortTag(tag)}</span>
				</label>
			{/each}
		</div>
	</div>
</div>
