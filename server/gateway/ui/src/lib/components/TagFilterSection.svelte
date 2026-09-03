<script lang="ts">
	import type { CatalogTag, SuggestedTag } from '$lib/types';
	import { ExtractorUnavailableError, extractTaxonomy, suggestTagsFromExtraction } from '$lib/api';
	import { onDestroy } from 'svelte';

	interface Props {
		catalogTags: CatalogTag[];
		tagsLoading?: boolean;
		activeTags: Set<string>;
		onToggleTag: (tagId: string, checked: boolean) => void;
	}

	let { catalogTags, tagsLoading = false, activeTags, onToggleTag }: Props = $props();

	let tagSearch = $state('');

	/**
	 * Suggestions for `suggestedFor`, which stays empty until the first
	 * extraction: an empty list means "matched nothing" only once it is set.
	 */
	let suggestions = $state<SuggestedTag[]>([]);
	let suggestedFor = $state('');
	let suggesting = $state(false);
	let suggestError = $state('');

	/**
	 * Discovered lazily from the first 503 rather than probed on mount: a probe
	 * would spend a model inference on every page load to answer a question that
	 * the first real use answers for free.
	 */
	let extractorAvailable = $state(true);

	let suggestAbort: AbortController | undefined;

	let query = $derived(tagSearch.trim());

	let visibleTags = $derived(
		query ? catalogTags.filter((t) => t.label.toLowerCase().includes(query.toLowerCase())) : catalogTags
	);

	/**
	 * Suggestions the substring list is not already showing. A tag that appears in
	 * both would otherwise render two checkboxes for the same filter.
	 */
	let newSuggestions = $derived.by(() => {
		const shown = new Set(visibleTags.map((t) => t.id));

		return suggestions.filter((s) => !shown.has(s.id));
	});

	let canSuggest = $derived(extractorAvailable && query.length > 0 && !suggesting);

	/**
	 * The path most users find this feature through: they typed a phrase, so
	 * substring matching found nothing.
	 */
	let promptForSuggestions = $derived(
		extractorAvailable &&
			query.length > 0 &&
			visibleTags.length === 0 &&
			!tagsLoading &&
			!suggesting &&
			!suggestedFor &&
			!suggestError
	);

	/**
	 * Keyed on `suggestions`, not `newSuggestions`: suggestions that all turned
	 * out to be already listed above are not "nothing found", and the list above
	 * is the answer.
	 */
	let noSuggestionsFound = $derived(
		!suggesting && suggestedFor !== '' && suggestions.length === 0 && !suggestError
	);

	/** Stale suggestions describe text the user has since edited away. */
	function clearSuggestions() {
		suggestAbort?.abort();
		suggestions = [];
		suggestedFor = '';
		suggestError = '';
	}

	async function runSuggest() {
		if (!canSuggest) return;

		suggestAbort?.abort();
		suggestAbort = new AbortController();
		const signal = suggestAbort.signal;

		suggesting = true;
		suggestError = '';
		const text = query;

		try {
			const extraction = await extractTaxonomy(text, signal);
			if (signal.aborted) return;

			suggestions = suggestTagsFromExtraction(extraction, catalogTags);
			suggestedFor = text;
		} catch (e) {
			if (signal.aborted) return;

			suggestions = [];
			suggestedFor = '';

			// No extractor is configured on this gateway, so there is nothing the
			// user can do about it. Withdraw the action instead of blaming them.
			if (e instanceof ExtractorUnavailableError) {
				extractorAvailable = false;

				return;
			}

			suggestError = e instanceof Error ? e.message : 'Unknown error';
		} finally {
			if (!signal.aborted) suggesting = false;
		}
	}

	onDestroy(() => {
		suggestAbort?.abort();
	});
</script>

{#snippet tagCheckbox(id: string, label: string, hint?: string)}
	<label class="flex items-start gap-2 text-sm text-ink cursor-pointer">
		<input
			type="checkbox"
			checked={activeTags.has(id)}
			onchange={(e) => onToggleTag(id, (e.target as HTMLInputElement).checked)}
			class="mt-0.5 rounded border-line-strong text-brand-500 focus:ring-brand-500"
		/>
		<span class="min-w-0">
			<span class="block truncate" title={id}>{label}</span>
			{#if hint}
				<span class="block truncate text-xs text-ink-weak" title={hint}>{hint}</span>
			{/if}
		</span>
	</label>
{/snippet}

<div class="flex-1 flex flex-col min-h-0">
	<div class="flex items-center justify-between gap-2 mb-2 flex-shrink-0">
		<span class="block text-xs font-semibold uppercase tracking-wide text-ink-medium">Tags</span>
		{#if tagsLoading}
			<span class="text-xs text-ink-weak">Loading…</span>
		{/if}
	</div>

	<input
		type="text"
		placeholder="Filter tags..."
		class="w-full rounded-control border-2 border-line bg-surface-light px-3 py-1.5 text-sm text-ink placeholder:text-ink-weak focus:outline-none focus:border-brand-500 mb-2 flex-shrink-0"
		bind:value={tagSearch}
		oninput={clearSuggestions}
		onkeydown={(e) => {
			if (e.key === 'Enter') runSuggest();
		}}
	/>

	{#if extractorAvailable}
		<button
			type="button"
			onclick={runSuggest}
			disabled={!canSuggest}
			aria-busy={suggesting}
			class="w-full rounded-control border-2 border-line bg-surface-light px-3 py-1.5 mb-2 flex-shrink-0 text-sm font-medium text-brand-600 transition hover:border-brand-500 hover:bg-brand-50 focus:outline-none focus:border-brand-500 disabled:cursor-not-allowed disabled:text-ink-weak disabled:hover:border-line disabled:hover:bg-surface-light"
		>
			{#if suggesting}
				<span class="inline-flex items-center gap-2">
					<span
						class="animate-spin rounded-full h-3.5 w-3.5 border-b-2 border-brand-500"
						aria-hidden="true"
					></span>
					Suggesting…
				</span>
			{:else}
				Suggest tags
			{/if}
		</button>
	{/if}

	<div class="space-y-1.5 overflow-y-auto flex-1">
		{#each visibleTags as tag (tag.id)}
			{@render tagCheckbox(tag.id, tag.label)}
		{/each}

		{#if visibleTags.length === 0 && query && !tagsLoading}
			<p class="text-sm text-ink-weak">
				No tags match that{#if promptForSuggestions}
					&nbsp;— try suggesting from your description{/if}.
			</p>
		{/if}

		{#if suggestError}
			<p class="text-sm text-red-600" role="alert">Could not suggest tags: {suggestError}</p>
		{/if}

		{#if newSuggestions.length > 0}
			<div class="pt-3 mt-3 border-t border-line">
				<span
					class="block text-xs font-semibold uppercase tracking-wide text-ink-medium mb-2"
					title="Taxonomy labels the extractor matched to your description, kept only where a record carries them"
				>
					Suggested from your description
				</span>
				<div class="space-y-1.5">
					{#each newSuggestions as suggestion (suggestion.id)}
						{@render tagCheckbox(suggestion.id, suggestion.label, suggestion.name)}
					{/each}
				</div>
			</div>
		{/if}

		{#if noSuggestionsFound}
			<p class="text-sm text-ink-weak pt-3 mt-3 border-t border-line">
				No taxonomy labels from your description match a tag any record here carries.
			</p>
		{/if}
	</div>
</div>
