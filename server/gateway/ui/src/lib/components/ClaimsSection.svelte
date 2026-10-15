<script lang="ts">
	import type { ClaimSummary } from '$lib/types';
	import { formatCheckedAt } from '$lib/utils';

	interface Props {
		claims: ClaimSummary[];
	}

	let { claims }: Props = $props();

	function roleLabel(role: string): string {
		return `${role.charAt(0).toUpperCase()}${role.slice(1)} claim`;
	}

	function statusPill(status: string): { label: string; classes: string } {
		switch (status) {
			case 'verified': return { label: 'Verified', classes: 'bg-emerald-100 text-emerald-700' };
			case 'failed': return { label: 'Failed', classes: 'bg-red-100 text-red-700' };
			default: return { label: status, classes: 'bg-surface-tag text-ink' };
		}
	}
</script>

{#if claims.length > 0}
	<div>
		<span class="font-semibold text-ink-strong text-sm mb-2 block">Claims</span>
		<div class="rounded-card border border-line p-3 space-y-1">
			{#each claims as claim (claim.role)}
				{@const pill = statusPill(claim.status)}
				{@const checkedAt = formatCheckedAt(claim.verifiedAt)}
				<div class="text-sm border-t border-line/70 pt-2 first:border-t-0 first:pt-0">
					<div class="flex items-center justify-between gap-2">
						<span class="font-medium text-ink">{roleLabel(claim.role)}</span>
						<div class="flex items-center gap-2">
							<span class="px-2 py-0.5 rounded-full text-xs font-semibold {pill.classes}">{pill.label}</span>
							{#if checkedAt}
								<span class="text-xs text-ink-weak">{checkedAt}</span>
							{/if}
						</div>
					</div>
					<p class="mt-0.5 text-xs font-mono break-all select-all text-ink-medium">{claim.subject}</p>
				</div>
			{/each}
		</div>
	</div>
{/if}
