<script lang="ts">
	import type { ScanManifest } from '$lib/types';

	interface Props {
		scan: ScanManifest;
	}

	let { scan }: Props = $props();

	function severityColorClass(sev: string): string {
		switch (sev) {
			case 'CRITICAL': return 'bg-red-100 text-red-700';
			case 'HIGH': return 'bg-orange-100 text-orange-700';
			case 'MEDIUM': return 'bg-amber-100 text-amber-700';
			case 'LOW': return 'bg-blue-100 text-blue-700';
			default: return 'bg-emerald-100 text-emerald-700';
		}
	}

	function toTitleCase(s: string): string {
		return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
	}

	let colorClass = $derived(severityColorClass(scan.maxSeverity));
	let hasFindings = $derived(
		!['NONE', '', 'INFO'].includes(scan.maxSeverity)
	);
	let title = $derived(
		scan.isSafe
			? hasFindings
				? `Security Scanned — Found at least one ${toTitleCase(scan.maxSeverity)} severity issue`
				: 'Security Scanned — No issues found'
			: hasFindings
				? `Not Safe — Found at least one ${toTitleCase(scan.maxSeverity)} severity issue`
				: 'Not Safe'
	);
</script>

<span
	class="inline-flex flex-shrink-0 items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold {colorClass}"
	{title}
>
	<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
		{#if !scan.isSafe}
			<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
		{:else if hasFindings}
			<path fill-rule="evenodd" d="M10 1.944A11.954 11.954 0 012.166 5C2.056 5.649 2 6.319 2 7c0 5.225 3.34 9.67 8 11.317C14.66 16.67 18 12.225 18 7c0-.682-.057-1.35-.166-2.001A11.954 11.954 0 0110 1.944zM11 14a1 1 0 11-2 0 1 1 0 012 0zm0-7a1 1 0 10-2 0v3a1 1 0 102 0V7z" clip-rule="evenodd"/>
		{:else}
			<path d="M10 1.944A11.954 11.954 0 012.166 5C2.056 5.649 2 6.319 2 7c0 5.225 3.34 9.67 8 11.317C14.66 16.67 18 12.225 18 7c0-.682-.057-1.35-.166-2.001A11.954 11.954 0 0110 1.944z"/>
		{/if}
	</svg>
</span>
