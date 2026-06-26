<script lang="ts">
	import type { AICardFilterCriteria } from '$lib/types';
	import { extractShortTag } from '$lib/utils';

	interface Props {
		allTags: string[];
		catalogHydrating?: boolean;
		onchange: (criteria: AICardFilterCriteria) => void;
	}

	let { allTags, catalogHydrating = false, onchange }: Props = $props();

	let searchQuery = $state('');
	let mediaTypes = $state<Set<string>>(new Set(['all']));
	let statusFilter = $state('all');
	let activeTags = $state<Set<string>>(new Set());
	let tagSearch = $state('');

	let visibleTags = $derived(
		tagSearch ? allTags.filter((t) => t.toLowerCase().includes(tagSearch.toLowerCase())) : allTags
	);

	function notifyChange() {
		onchange({ searchQuery, mediaTypes, statusFilter, activeTags });
	}

	function ensureMediaTypesSelection(next: Set<string>) {
		if (next.size === 0) {
			next.add('all');
		}

		return next;
	}

	function leafValues(opt: MediaTypeOption): string[] {
		if (isMediaTypeGroup(opt)) {
			return opt.children.flatMap((child) => leafValues(child));
		}

		return [opt.value];
	}

	type GroupCheckState = 'none' | 'some' | 'all';

	function groupCheckState(group: MediaTypeGroup): GroupCheckState {
		const values = leafValues(group);
		if (values.length === 0) {
			return 'none';
		}

		const selected = values.filter((value) => mediaTypes.has(value)).length;
		if (selected === 0) {
			return 'none';
		}
		if (selected === values.length) {
			return 'all';
		}

		return 'some';
	}

	function handleMediaType(value: string, checked: boolean) {
		const next = new Set(mediaTypes);
		if (value === 'all' && checked) {
			next.clear();
			next.add('all');
		} else if (value !== 'all') {
			next.delete('all');
			if (checked) next.add(value);
			else next.delete(value);
		}
		mediaTypes = ensureMediaTypesSelection(next);
		notifyChange();
	}

	function handleGroupMediaType(group: MediaTypeGroup, checked: boolean) {
		const next = new Set(mediaTypes);
		next.delete('all');

		for (const value of leafValues(group)) {
			if (checked) {
				next.add(value);
			} else {
				next.delete(value);
			}
		}

		mediaTypes = ensureMediaTypesSelection(next);
		notifyChange();
	}

	/** Keeps parent checkboxes in the indeterminate (−) state when partially selected. */
	function indeterminateCheckbox(node: HTMLInputElement, indeterminate: boolean) {
		node.indeterminate = indeterminate;
		return {
			update(next: boolean) {
				node.indeterminate = next;
			}
		};
	}

	function handleTag(tag: string, checked: boolean) {
		const next = new Set(activeTags);
		if (checked) next.add(tag);
		else next.delete(tag);
		activeTags = next;
		notifyChange();
	}

	type MediaTypeLeaf = {
		value: string;
		label: string;
	};

	type MediaTypeGroup = {
		label: string;
		children: MediaTypeOption[];
	};

	type MediaTypeOption = MediaTypeLeaf | MediaTypeGroup;

	function isMediaTypeGroup(opt: MediaTypeOption): opt is MediaTypeGroup {
		return 'children' in opt;
	}

	/** Top-level media type filters; groups may nest further checkboxes under `children`. */
	const MEDIA_TYPE_OPTIONS: MediaTypeOption[] = [
		{ value: 'all', label: 'All' },
		{ value: 'application/a2a-agent-card+json', label: 'A2A Agent' },
		{ value: 'application/mcp-server-card+json', label: 'MCP Server' },
        {
            label: 'SKILL',
            children: [
                { value: 'application/agent-skills+md', label: 'Markdown' },
                { value: 'application/agent-skills+gzip', label: 'Bundle' }
            ]
        }
	];
</script>

{#snippet mediaTypeOption(opt: MediaTypeOption)}
	{#if isMediaTypeGroup(opt)}
		{@const groupState = groupCheckState(opt)}
		<label class="flex items-center gap-2 text-sm text-ink cursor-pointer">
			<input
				type="checkbox"
				checked={groupState === 'all'}
				use:indeterminateCheckbox={groupState === 'some'}
				onchange={(e) =>
					handleGroupMediaType(opt, (e.target as HTMLInputElement).checked)}
				class="rounded border-line-strong text-brand-500 focus:ring-brand-500"
			/>
			<span>{opt.label}</span>
		</label>
		<div class="space-y-1.5 pl-5 ml-0.5 border-l-2 border-line/70">
			{#each opt.children as child (child.label)}
				{@render mediaTypeOption(child)}
			{/each}
		</div>
	{:else}
		<label class="flex items-center gap-2 text-sm text-ink cursor-pointer">
			<input
				type="checkbox"
				checked={mediaTypes.has(opt.value)}
				onchange={(e) => handleMediaType(opt.value, (e.target as HTMLInputElement).checked)}
				class="rounded border-line-strong text-brand-500 focus:ring-brand-500"
			/>
			<span>{opt.label}</span>
		</label>
	{/if}
{/snippet}

<div class="bg-surface-strong rounded-card border border-line p-4 space-y-5 sticky top-24 max-h-[calc(100vh-7rem)] flex flex-col overflow-hidden">
	<div class="flex-shrink-0">
		<label for="search" class="block text-xs font-semibold uppercase tracking-wide text-ink-medium mb-1.5">Search</label>
		<input
			type="text"
			id="search"
			placeholder="Filter by name..."
			class="w-full rounded-control border-2 border-line bg-surface-light px-3 py-2 text-sm text-ink placeholder:text-ink-weak focus:outline-none focus:border-brand-500"
			bind:value={searchQuery}
			oninput={notifyChange}
		/>
	</div>

	<div class="flex-shrink-0">
		<span class="block text-xs font-semibold uppercase tracking-wide text-ink-medium mb-2">Media Type</span>
		<div class="space-y-1.5">
			{#each MEDIA_TYPE_OPTIONS as opt (opt.label)}
				{@render mediaTypeOption(opt)}
			{/each}
		</div>
	</div>

	<div class="flex-shrink-0">
		<span class="block text-xs font-semibold uppercase tracking-wide text-ink-medium mb-2">Status</span>
		<div class="space-y-1.5">
			{#each ['all', 'trusted', 'verified'] as value}
				<label class="flex items-center gap-2 text-sm text-ink cursor-pointer">
					<input
						type="radio"
						name="status"
						{value}
						checked={statusFilter === value}
						onchange={() => {
							statusFilter = value;
							notifyChange();
						}}
						class="border-line-strong text-brand-500 focus:ring-brand-500"
					/>
					<span class="capitalize">{value}</span>
				</label>
			{/each}
		</div>
	</div>

	<div class="flex-1 flex flex-col min-h-0">
		<div class="flex items-center justify-between gap-2 mb-2 flex-shrink-0">
			<span class="block text-xs font-semibold uppercase tracking-wide text-ink-medium">Tags</span>
			{#if catalogHydrating}
				<span class="text-xs text-ink-weak">Loading…</span>
			{/if}
		</div>
		<input
			type="text"
			placeholder="Filter tags..."
			class="w-full rounded-control border-2 border-line bg-surface-light px-3 py-1.5 text-sm text-ink placeholder:text-ink-weak focus:outline-none focus:border-brand-500 mb-2 flex-shrink-0"
			bind:value={tagSearch}
		/>
		<div class="space-y-1.5 overflow-y-auto flex-1">
			{#each visibleTags as tag}
				<label class="flex items-center gap-2 text-sm text-ink cursor-pointer">
					<input
						type="checkbox"
						checked={activeTags.has(tag)}
						onchange={(e) => handleTag(tag, (e.target as HTMLInputElement).checked)}
						class="rounded border-line-strong text-brand-500 focus:ring-brand-500"
					/>
					<span class="truncate" title={tag}>{extractShortTag(tag)}</span>
				</label>
			{/each}
		</div>
	</div>
</div>
