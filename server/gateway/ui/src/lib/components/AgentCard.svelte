<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { extractEntryTypes, extractShortTag, hasTrustManifest, fakeStats, formatDownloads } from '$lib/utils';
	import MediaTypeBadge from './MediaTypeBadge.svelte';
	import StarRating from './StarRating.svelte';
	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Props {
		agent: CatalogEntry;
		onclick: () => void;
	}

	let { agent, onclick }: Props = $props();

	let types = $derived(extractEntryTypes(agent));
	let verified = $derived(hasTrustManifest(agent));
	let tags = $derived((agent.tags || []).slice(0, 3));
	let stats = $derived(fakeStats(agent));
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
<article
	class="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow cursor-pointer flex flex-col h-full"
	onclick={onclick}
	onkeydown={(e) => { if (e.key === 'Enter') onclick(); }}
	role="button"
	tabindex="0"
>
	<div class="flex items-start justify-between gap-3 mb-2">
		<div class="flex items-center gap-1.5 min-w-0">
			<h3 class="font-semibold text-gray-900 truncate">{agent.displayName || 'Unnamed Agent'}</h3>
			{#if verified}
				<VerifiedBadge />
			{/if}
		</div>
		<span class="text-xs text-gray-400 whitespace-nowrap">{agent.version || ''}</span>
	</div>

	<p class="text-sm text-gray-600 mb-3 line-clamp-2">{agent.description || 'No description available.'}</p>

	<div class="flex flex-wrap gap-1.5 mb-3">
		{#each types as type}
			<MediaTypeBadge {type} />
		{/each}
	</div>

	<div class="flex flex-wrap gap-1.5 mb-3">
		{#each tags as tag}
			<span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-600 truncate max-w-[150px]" title={tag}>
				{extractShortTag(tag)}
			</span>
		{/each}
	</div>

	<div class="flex items-center justify-between text-xs text-gray-400 mt-auto pt-2 border-t border-gray-100">
		<StarRating rating={stats.rating} />
		<span class="flex items-center gap-3">
			<span class="inline-flex items-center gap-1" title="Downloads">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
				{formatDownloads(stats.downloads)}
			</span>
			<span class="inline-flex items-center gap-1" title="Providers">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/></svg>
				{stats.providers}
			</span>
		</span>
	</div>
</article>
