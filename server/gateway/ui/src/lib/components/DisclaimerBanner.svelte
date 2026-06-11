<script lang="ts">
	import { onMount } from 'svelte';

	const STORAGE_KEY = 'ai-catalog-disclaimer-dismissed';

	let visible = $state(false);

	onMount(() => {
		visible = localStorage.getItem(STORAGE_KEY) !== 'true';
	});

	function dismiss() {
		visible = false;
		localStorage.setItem(STORAGE_KEY, 'true');
	}
</script>

{#if visible}
	<div class="bg-amber-50 border-b border-amber-200" role="status">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3 flex items-start gap-3">
			<svg
				class="w-5 h-5 text-amber-600 flex-shrink-0 mt-0.5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
			<p class="text-sm text-amber-900 flex-1">
				<span class="font-medium">Preview data.</span>
				Star ratings, download counts, and provider statistics shown on catalog entries are
				placeholder values for demonstration only and do not reflect real usage or federation data.
			</p>
			<button
				type="button"
				class="p-1 rounded-md text-amber-700 hover:bg-amber-100 transition flex-shrink-0"
				onclick={dismiss}
				aria-label="Dismiss disclaimer"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		</div>
	</div>
{/if}
