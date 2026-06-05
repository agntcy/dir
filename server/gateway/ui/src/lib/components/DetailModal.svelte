<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { extractEntryTypes, extractShortTag, hasTrustManifest, fakeStats, extractCid, exportFormatForType, extractEntryName, extractEntryVersion } from '$lib/utils';
	import MediaTypeBadge from './MediaTypeBadge.svelte';
	import StarRating from './StarRating.svelte';
	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Props {
		agent: CatalogEntry;
		onclose: () => void;
	}

	let { agent, onclose }: Props = $props();

	let stats = $derived(fakeStats(agent));
	let verified = $derived(hasTrustManifest(agent));
	let tags = $derived(agent.tags || []);
	let cid = $derived(extractCid(agent.identifier));
	let entries = $derived(agent.data?.entries || []);
	let isSingleModule = $derived(!entries.length && agent.mediaType !== 'application/ai-catalog+json');
	let jsonStr = $derived(JSON.stringify(agent, null, 2));

	let copied = $state(false);

	function copyJson() {
		navigator.clipboard.writeText(jsonStr).then(() => {
			copied = true;
			setTimeout(() => { copied = false; }, 2000);
		});
	}

	function handleBackdrop(e: MouseEvent) {
		if (e.target === e.currentTarget) onclose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm" onclick={handleBackdrop}>
	<div class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col overflow-hidden">
		<div class="flex items-start justify-between px-6 py-4 border-b border-gray-200">
			<div class="min-w-0 flex-1 mr-4">
				<div class="flex items-center gap-2">
					<h2 class="text-lg font-semibold text-gray-900 truncate">{agent.displayName || 'Unnamed Agent'}</h2>
					{#if verified}
						<VerifiedBadge />
					{/if}
				</div>
				<p class="text-sm text-gray-500 mt-0.5 break-all select-all cursor-text">{agent.identifier || ''}</p>
			</div>
			<button class="p-1.5 rounded-lg hover:bg-gray-100 transition flex-shrink-0" onclick={onclose} aria-label="Close">
				<svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
				</svg>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto px-6 py-5 space-y-5">
			<p class="text-sm text-gray-600">{agent.description || 'No description available.'}</p>

			<div class="grid grid-cols-2 sm:grid-cols-4 gap-x-6 gap-y-3 text-sm border rounded-lg p-4 bg-gray-50">
				<div>
					<span class="font-medium text-gray-500 text-xs uppercase tracking-wide">Version</span>
					<p class="text-gray-900 mt-0.5 font-medium">{agent.version || 'N/A'}</p>
				</div>
				<div>
					<span class="font-medium text-gray-500 text-xs uppercase tracking-wide">Updated</span>
					<p class="text-gray-900 mt-0.5 font-medium">{agent.updatedAt ? new Date(agent.updatedAt).toLocaleDateString() : 'N/A'}</p>
				</div>
				<div>
					<span class="font-medium text-gray-500 text-xs uppercase tracking-wide">Downloads</span>
					<p class="text-gray-900 mt-0.5 font-medium">{stats.downloads.toLocaleString()}</p>
				</div>
				<div>
					<span class="font-medium text-gray-500 text-xs uppercase tracking-wide">Providers</span>
					<p class="text-gray-900 mt-0.5 font-medium">{stats.providers} node{stats.providers !== 1 ? 's' : ''}</p>
				</div>
			</div>

			<div class="flex items-center gap-2 text-sm">
				<span class="font-medium text-gray-500 text-xs uppercase tracking-wide">Rating</span>
				<StarRating rating={stats.rating} />
				<span class="text-gray-700 font-medium">{stats.rating.toFixed(1)}</span>
			</div>

			<!-- Entries table -->
			<div>
				<span class="font-medium text-gray-700 text-sm mb-2 block">Entries</span>
				{#if entries.length > 0}
					<table class="w-full text-left">
						<thead>
							<tr class="text-xs text-gray-500 uppercase tracking-wide">
								<th class="pb-2 font-medium">Name</th>
								<th class="pb-2 font-medium">Version</th>
								<th class="pb-2 font-medium">Type</th>
								<th class="pb-2 font-medium text-right">Export</th>
							</tr>
						</thead>
						<tbody>
						{#each entries as entry}
							{@const exp = exportFormatForType(entry.mediaType || '')}
							{@const name = extractEntryName(entry)}
							{@const version = extractEntryVersion(entry)}
							{@const filename = name.replace(/[^a-z0-9_-]/gi, '_') + '.' + exp.ext}
							<tr class="border-t border-gray-100">
								<td class="py-2 pr-3 text-sm text-gray-900">{name}</td>
								<td class="py-2 pr-3 text-sm text-gray-500">{version}</td>
								<td class="py-2 pr-3"><MediaTypeBadge type={entry.mediaType || ''} /></td>
									<td class="py-2 text-right">
										<a href="/v1/agents/{encodeURIComponent(cid)}/export?format={exp.format}" download={filename}
											class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium text-brand-700 bg-brand-50 rounded-lg hover:bg-brand-100 transition">
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
											{exp.label}
										</a>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
			{:else if isSingleModule}
				{@const exp = exportFormatForType(agent.mediaType)}
				{@const name = agent.data?.skillManifest?.name || agent.displayName || 'Unnamed'}
				{@const version = agent.data?.skillManifest?.version || agent.version || '-'}
				{@const filename = name.replace(/[^a-z0-9_-]/gi, '_') + '.' + exp.ext}
					<table class="w-full text-left">
						<thead>
							<tr class="text-xs text-gray-500 uppercase tracking-wide">
								<th class="pb-2 font-medium">Name</th>
								<th class="pb-2 font-medium">Version</th>
								<th class="pb-2 font-medium">Type</th>
								<th class="pb-2 font-medium text-right">Export</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-t border-gray-100">
								<td class="py-2 pr-3 text-sm text-gray-900">{name}</td>
								<td class="py-2 pr-3 text-sm text-gray-500">{version}</td>
								<td class="py-2 pr-3"><MediaTypeBadge type={agent.mediaType} /></td>
								<td class="py-2 text-right">
									<a href="/v1/agents/{encodeURIComponent(cid)}/export?format={exp.format}" download={filename}
										class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium text-brand-700 bg-brand-50 rounded-lg hover:bg-brand-100 transition">
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
										{exp.label}
									</a>
								</td>
							</tr>
						</tbody>
					</table>
				{:else}
					<span class="text-sm text-gray-400">No entries</span>
				{/if}
			</div>

			<!-- Tags -->
			<div>
				<span class="font-medium text-gray-700 text-sm">Tags</span>
				<div class="flex flex-wrap gap-1.5 mt-1.5">
					{#each tags as tag}
						<span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs bg-gray-100 text-gray-700 break-all">{tag}</span>
					{:else}
						<span class="text-sm text-gray-400">None</span>
					{/each}
				</div>
			</div>

			<!-- Trust manifest -->
		{#if agent.trustManifest?.signature}
			{@const tm = agent.trustManifest}
			{@const shortSig = tm.signature && tm.signature.length > 40 ? tm.signature.slice(0, 20) + '...' + tm.signature.slice(-20) : tm.signature}
				<div class="rounded-lg border border-blue-200 bg-blue-50 p-4">
					<div class="flex items-center gap-2 mb-3">
						<VerifiedBadge />
						<span class="font-medium text-blue-800 text-sm">Trust Manifest</span>
					</div>
					<div class="space-y-2 text-sm">
						<div>
							<span class="font-medium text-blue-900">Identity</span>
							<p class="text-blue-700 mt-0.5 break-all text-xs font-mono">{tm.identity || 'Unknown'}</p>
						</div>
						<div class="grid grid-cols-3 gap-3">
							<div>
								<span class="font-medium text-blue-900">Type</span>
								<p class="text-blue-700 mt-0.5">{tm.identityType || 'Unknown'}</p>
							</div>
							<div>
								<span class="font-medium text-blue-900">Attestations</span>
								<p class="text-blue-700 mt-0.5">{(tm.attestations || []).length}</p>
							</div>
							<div>
								<span class="font-medium text-blue-900">Provenance</span>
								<p class="text-blue-700 mt-0.5">{(tm.provenance || []).length}</p>
							</div>
						</div>
						<div>
							<span class="font-medium text-blue-900">Signature</span>
							<p class="text-blue-700 mt-0.5 text-xs font-mono break-all select-all cursor-text" title={tm.signature}>{shortSig}</p>
						</div>
					</div>
				</div>
			{/if}

			<!-- JSON -->
			<div>
				<div class="flex items-center justify-between mb-2">
					<span class="font-medium text-gray-700 text-sm">Catalog Entry JSON</span>
					<button onclick={copyJson} class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-brand-700 bg-brand-50 rounded-lg hover:bg-brand-100 transition">
						{#if copied}
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
							Copied!
						{:else}
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
							Copy
						{/if}
					</button>
				</div>
				<pre class="bg-gray-900 text-gray-100 rounded-lg p-4 text-xs overflow-x-auto max-h-64 overflow-y-auto"><code>{jsonStr}</code></pre>
			</div>
		</div>
	</div>
</div>
