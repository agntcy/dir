<script lang="ts">
	import '../app.css';
	import CatalogHeader from '$lib/components/CatalogHeader.svelte';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<CatalogHeader />

{@render children()}
