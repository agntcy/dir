<script lang="ts">
	import '../app.css';
	import CatalogHeader from '$lib/components/CatalogHeader.svelte';
	import { catalogTitleState } from '$lib/catalog-title.svelte';
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

	onMount(() => {
		catalogTitleState.load();
	});
</script>

<svelte:head>
	<title>{catalogTitleState.title}</title>
</svelte:head>

<CatalogHeader />

{@render children()}
