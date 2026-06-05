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
			class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed transition"
			disabled={currentPage <= 1}
			onclick={() => onpage(currentPage - 1)}
		>Previous</button>

		<div class="flex items-center gap-1">
			{#each pages as p}
				{#if p === '...'}
					<span class="px-2 py-1 text-sm text-gray-400">...</span>
				{:else}
					<button
						class="px-3 py-1.5 text-sm rounded-lg transition {p === currentPage ? 'bg-brand-600 text-white' : 'border border-gray-300 hover:bg-gray-100'}"
						onclick={() => onpage(p)}
					>{p}</button>
				{/if}
			{/each}
		</div>

		<button
			class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed transition"
			disabled={currentPage >= totalPages}
			onclick={() => onpage(currentPage + 1)}
		>Next</button>
	</nav>
{/if}
