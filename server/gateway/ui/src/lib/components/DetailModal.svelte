<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { extractEntryTypes, extractShortTag, hasTrustManifest, getScanManifest, getUsageMetrics, formatDownloads, extractCid, exportFormatForType, extractEntryName, extractEntryVersion } from '$lib/utils';
	import MediaTypeBadge from './MediaTypeBadge.svelte';

	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Props {
		aicard: CatalogEntry;
		onclose: () => void;
	}

	let { aicard, onclose }: Props = $props();

	let metrics = $derived(getUsageMetrics(aicard));
	let verified = $derived(hasTrustManifest(aicard));
	let scanManifest = $derived(getScanManifest(aicard));
	let tags = $derived(aicard.tags || []);
	let cid = $derived(extractCid(aicard.identifier));
	let entries = $derived(aicard.data?.entries || []);
	let isSingleModule = $derived(!entries.length && aicard.mediaType !== 'application/ai-catalog+json');
	let jsonStr = $derived(JSON.stringify(aicard, null, 2));

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
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[rgba(209,219,246,0.4)]" onclick={handleBackdrop}>
	<div class="bg-surface-light rounded-card shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col overflow-hidden border border-line">
		<div class="flex items-start justify-between px-6 py-4 border-b border-line">
			<div class="min-w-0 flex-1 mr-4">
				<div class="flex items-center gap-2">
					<h2 class="text-lg font-semibold text-ink-strong truncate">{aicard.displayName || 'Unnamed AI card'}</h2>
					{#if verified}
						<VerifiedBadge />
					{/if}
				</div>
				<p class="text-sm text-ink-weak mt-0.5 break-all select-all cursor-text">{aicard.identifier || ''}</p>
			</div>
			<button class="p-1.5 rounded hover:bg-surface-strong transition flex-shrink-0" onclick={onclose} aria-label="Close">
				<svg class="w-5 h-5 text-ink-medium" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
				</svg>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto px-6 py-5 space-y-5">
			<p class="text-sm text-ink">{aicard.description || 'No description available.'}</p>

			<div class="grid grid-cols-2 sm:grid-cols-4 gap-x-6 gap-y-3 text-sm border border-line rounded-card p-4 bg-surface-strong">
				<div>
					<span class="font-semibold text-ink-medium text-xs uppercase tracking-wide">Version</span>
					<p class="text-ink-strong mt-0.5 font-medium">{aicard.version || 'N/A'}</p>
				</div>
				<div>
					<span class="font-semibold text-ink-medium text-xs uppercase tracking-wide">Updated</span>
					<p class="text-ink-strong mt-0.5 font-medium">{aicard.updatedAt ? new Date(aicard.updatedAt).toLocaleDateString() : 'N/A'}</p>
				</div>
				<div>
					<span class="font-semibold text-ink-medium text-xs uppercase tracking-wide">Pulls</span>
					<p class="text-ink-strong mt-0.5 font-medium">{metrics ? formatDownloads(metrics.pullCount) : '—'}</p>
				</div>
				<div>
					<span class="font-semibold text-ink-medium text-xs uppercase tracking-wide">Providers</span>
					<p class="text-ink-strong mt-0.5 font-medium">{metrics ? `${metrics.providerCount} node${metrics.providerCount !== 1 ? 's' : ''}` : '—'}</p>
				</div>
			</div>

			<!-- Entries table -->
			<div>
				<span class="font-semibold text-ink-strong text-sm mb-2 block">Entries</span>
				{#if entries.length > 0}
					<table class="w-full text-left">
						<thead>
							<tr class="text-xs text-ink-medium uppercase tracking-wide">
								<th class="pb-2 font-semibold">Name</th>
								<th class="pb-2 font-semibold">Version</th>
								<th class="pb-2 font-semibold">Type</th>
								<th class="pb-2 font-semibold text-right">Export</th>
							</tr>
						</thead>
						<tbody>
						{#each entries as entry}
							{@const exp = exportFormatForType(entry.mediaType || '')}
							{@const name = extractEntryName(entry)}
							{@const version = extractEntryVersion(entry)}
							{@const filename = name.replace(/[^a-z0-9_-]/gi, '_') + '.' + exp.ext}
							<tr class="border-t border-line/70">
								<td class="py-2 pr-3 text-sm text-ink-strong">{name}</td>
								<td class="py-2 pr-3 text-sm text-ink-medium">{version}</td>
								<td class="py-2 pr-3"><MediaTypeBadge type={entry.mediaType || ''} /></td>
									<td class="py-2 text-right">
										<a href="/v1/agents/{encodeURIComponent(cid)}/export?format={exp.format}" download={filename}
											class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-brand-600 bg-brand-200 rounded hover:bg-brand-300 transition">
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
											{exp.label}
										</a>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
			{:else if isSingleModule}
				{@const exp = exportFormatForType(aicard.mediaType)}
				{@const name = aicard.data?.skillManifest?.name || aicard.displayName || 'Unnamed'}
				{@const version = aicard.data?.skillManifest?.version || aicard.version || '-'}
				{@const filename = name.replace(/[^a-z0-9_-]/gi, '_') + '.' + exp.ext}
					<table class="w-full text-left">
						<thead>
							<tr class="text-xs text-ink-medium uppercase tracking-wide">
								<th class="pb-2 font-semibold">Name</th>
								<th class="pb-2 font-semibold">Version</th>
								<th class="pb-2 font-semibold">Type</th>
								<th class="pb-2 font-semibold text-right">Export</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-t border-line/70">
								<td class="py-2 pr-3 text-sm text-ink-strong">{name}</td>
								<td class="py-2 pr-3 text-sm text-ink-medium">{version}</td>
								<td class="py-2 pr-3"><MediaTypeBadge type={aicard.mediaType} /></td>
								<td class="py-2 text-right">
									<a href="/v1/agents/{encodeURIComponent(cid)}/export?format={exp.format}" download={filename}
										class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-brand-600 bg-brand-200 rounded hover:bg-brand-300 transition">
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
										{exp.label}
									</a>
								</td>
							</tr>
						</tbody>
					</table>
				{:else}
					<span class="text-sm text-ink-weak">No entries</span>
				{/if}
			</div>

			<!-- Tags -->
			<div>
				<span class="font-semibold text-ink-strong text-sm">Tags</span>
				<div class="flex flex-wrap gap-1.5 mt-1.5">
					{#each tags as tag}
						<span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold bg-surface-tag text-ink break-all">{tag}</span>
					{:else}
						<span class="text-sm text-ink-weak">None</span>
					{/each}
				</div>
			</div>

			<!-- Trust manifest -->
		{#if aicard.trustManifest?.signature}
			{@const tm = aicard.trustManifest}
			{@const shortSig = tm.signature && tm.signature.length > 40 ? tm.signature.slice(0, 20) + '...' + tm.signature.slice(-20) : tm.signature}
				<div class="rounded-card border border-line bg-brand-200 p-4">
					<div class="flex items-center gap-2 mb-3">
						<VerifiedBadge />
						<span class="font-semibold text-brand-800 text-sm">Trust Manifest</span>
					</div>
					<div class="space-y-2 text-sm">
						<div>
							<span class="font-semibold text-brand-800">Identity</span>
							<p class="text-brand-600 mt-0.5 break-all text-xs font-mono">{tm.identity || 'Unknown'}</p>
						</div>
						<div class="grid grid-cols-3 gap-3">
							<div>
								<span class="font-semibold text-brand-800">Type</span>
								<p class="text-brand-600 mt-0.5">{tm.identityType || 'Unknown'}</p>
							</div>
							<div>
								<span class="font-semibold text-brand-800">Attestations</span>
								<p class="text-brand-600 mt-0.5">{(tm.attestations || []).length}</p>
							</div>
							<div>
								<span class="font-semibold text-brand-800">Provenance</span>
								<p class="text-brand-600 mt-0.5">{(tm.provenance || []).length}</p>
							</div>
						</div>
						<div>
							<span class="font-semibold text-brand-800">Signature</span>
							<p class="text-brand-600 mt-0.5 text-xs font-mono break-all select-all cursor-text" title={tm.signature}>{shortSig}</p>
						</div>
					</div>
				</div>
			{/if}

			<!-- Security scan -->
			{#if scanManifest}
				{@const sm = scanManifest}
				<div>
					<span class="font-semibold text-ink-strong text-sm mb-2 block">Scanner Details</span>
					<div class="rounded-card border border-line p-3 space-y-1">
						{#each sm.reports as report}
							{@const sev = report.maxSeverity || 'NONE'}
							{@const sevLabel = sev.charAt(0) + sev.slice(1).toLowerCase()}
							{@const sevClass = sev === 'CRITICAL' ? 'text-red-600' : sev === 'HIGH' ? 'text-orange-600' : sev === 'MEDIUM' ? 'text-amber-600' : sev === 'LOW' ? 'text-blue-600' : 'text-emerald-600'}
							<div class="flex items-center justify-between text-sm border-t border-line/70 pt-2 first:border-t-0 first:pt-0">
								<span class="font-medium text-ink capitalize">{report.scannerType.toLowerCase()} scanner</span>
								<div class="flex items-center gap-2">
									{#if report.isSafe}
										<span class="font-semibold text-xs">
											<span class="text-emerald-600">Safe</span>
											<span class="text-ink-weak mx-0.5">&middot;</span>
											<span class="{sevClass}">{sevLabel} severity</span>
										</span>
									{:else}
										<span class="font-semibold text-xs">
											<span class="text-red-600">Issues</span>
											<span class="text-ink-weak mx-0.5">&middot;</span>
											<span class="{sevClass}">{sevLabel} severity</span>
										</span>
									{/if}
									{#if report.updatedAt}
										<span class="text-xs text-ink-weak">{new Date(report.updatedAt).toLocaleDateString()}</span>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- JSON -->
			<div>
				<div class="flex items-center justify-between mb-2">
					<span class="font-semibold text-ink-strong text-sm">Catalog Entry JSON</span>
					<button onclick={copyJson} class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-brand-600 bg-brand-200 rounded hover:bg-brand-300 transition">
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
