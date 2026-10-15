<script lang="ts">
	import type { CatalogEntry } from '$lib/types';
	import { getIdentityClaim, getScanManifest, getTrustStatus } from '$lib/utils';
	import IdentityBadge from './IdentityBadge.svelte';
	import ScanBadge from './ScanBadge.svelte';
	import TrustedBadge from './TrustedBadge.svelte';
	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Props {
		aicard: CatalogEntry;
	}

	let { aicard }: Props = $props();

	let trustStatus = $derived(getTrustStatus(aicard));
	let identityClaim = $derived(getIdentityClaim(aicard));
	let scanManifest = $derived(getScanManifest(aicard));
</script>

{#if trustStatus?.trusted}
	<TrustedBadge />
{/if}
{#if trustStatus?.verified}
	<VerifiedBadge />
{/if}
{#if identityClaim}
	<IdentityBadge status={identityClaim.status} subject={identityClaim.subject} verifiedAt={identityClaim.verifiedAt} />
{/if}
{#if scanManifest}
	<ScanBadge scan={scanManifest} />
{/if}
