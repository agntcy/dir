<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { extractEntryTypes, extractShortTag, hasTrustManifest, getScanManifest, getUsageMetrics, formatDownloads } from '$lib/utils';
	import MediaTypeBadge from './MediaTypeBadge.svelte';
	import ScanBadge from './ScanBadge.svelte';
	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Props {
		aicard: CatalogEntry;
		onclick: () => void;
	}

	let { aicard, onclick }: Props = $props();

	let types = $derived(extractEntryTypes(aicard));
	let verified = $derived(hasTrustManifest(aicard));
	let scanManifest = $derived(getScanManifest(aicard));
	let tags = $derived((aicard.tags || []).slice(0, 3));
	let metrics = $derived(getUsageMetrics(aicard));
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
<article
	class="bg-surface-light rounded-card border border-line/70 shadow-card p-4 hover:shadow-card-hover hover:border-line-strong transition cursor-pointer flex flex-col h-full gap-3"
	onclick={onclick}
	onkeydown={(e) => { if (e.key === 'Enter') onclick(); }}
	role="button"
	tabindex="0"
>
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-1.5 min-w-0">
				<h3 class="font-semibold text-ink-strong truncate">{aicard.displayName || 'Unnamed AI card'}</h3>
				{#if verified}
					<VerifiedBadge />
				{/if}
				{#if scanManifest}
					<ScanBadge scan={scanManifest} iconOnly />
				{/if}
			</div>
			<p class="text-xs text-ink-medium mt-0.5">
				{#if aicard.version}Version {aicard.version}{/if}
				{#if aicard.version && aicard.updatedAt} &bull; {/if}
				{#if aicard.updatedAt}{new Date(aicard.updatedAt).toLocaleDateString()}{/if}
			</p>
		</div>
	</div>

	<p class="text-sm text-ink line-clamp-2">{aicard.description || 'No description available.'}</p>

	{#if tags.length}
		<div class="flex flex-wrap gap-1.5">
			{#each tags as tag}
				<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-surface-tag text-ink whitespace-normal break-words max-w-full" title={tag}>
					{extractShortTag(tag)}
				</span>
			{/each}
		</div>
	{/if}

	<div class="flex flex-wrap gap-1.5">
		{#each types as type}
			<MediaTypeBadge {type} />
		{/each}
	</div>

	{#if metrics}
		<div class="flex items-center justify-end gap-3 text-xs text-ink-weak mt-auto pt-2.5 border-t border-line/70">
			<span class="inline-flex items-center gap-1" title="Pulls">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
				{formatDownloads(metrics.pullCount)}
			</span>
			<span class="inline-flex items-center gap-1" title="Providers">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/></svg>
				{metrics.providerCount}
			</span>
		</div>
	{/if}
</article>
