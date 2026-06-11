<script lang="ts">
	interface Props {
		currentPage: number;
		totalPages: number;
		onpage: (page: number) => void;
	}

	let { currentPage, totalPages, onpage }: Props = $props();

	let pages = $derived.by(() => {
		const maxVisible = 7;
		const result: (number | '...')[] = [];
		if (totalPages <= maxVisible) {
			for (let i = 1; i <= totalPages; i++) result.push(i);
		} else {
			result.push(1);
			if (currentPage > 3) result.push('...');
			const start = Math.max(2, currentPage - 1);
			const end = Math.min(totalPages - 1, currentPage + 1);
			for (let i = start; i <= end; i++) result.push(i);
			if (currentPage < totalPages - 2) result.push('...');
			result.push(totalPages);
		}
		return result;
	});
</script>

{#if totalPages > 1}
	<nav class="flex items-center justify-center gap-2 mt-6 pb-4">
		<button
			class="px-3 py-1.5 text-sm font-medium rounded text-ink border-2 border-line bg-surface-light hover:border-line-strong disabled:opacity-40 disabled:cursor-not-allowed transition"
			disabled={currentPage <= 1}
			onclick={() => onpage(currentPage - 1)}
		>Previous</button>

		<div class="flex items-center gap-1">
			{#each pages as p}
				{#if p === '...'}
					<span class="px-2 py-1 text-sm text-ink-weak">...</span>
				{:else}
					<button
						class="min-w-9 h-9 px-2 text-sm font-medium rounded-full transition {p === currentPage ? 'bg-brand-500 text-white' : 'text-ink hover:bg-surface-strong'}"
						onclick={() => onpage(p)}
					>{p}</button>
				{/if}
			{/each}
		</div>

		<button
			class="px-3 py-1.5 text-sm font-medium rounded text-ink border-2 border-line bg-surface-light hover:border-line-strong disabled:opacity-40 disabled:cursor-not-allowed transition"
			disabled={currentPage >= totalPages}
			onclick={() => onpage(currentPage + 1)}
		>Next</button>
	</nav>
{/if}
