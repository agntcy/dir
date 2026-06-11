<script lang="ts">
	interface Props {
		type: string;
	}

	let { type }: Props = $props();

	let label = $derived(
		type.includes('a2a') ? 'A2A' :
		type.includes('mcp') ? 'MCP' :
		type.includes('agentskill') ? 'SKILL' :
		type.split('/').pop()?.split('+')[0] || type
	);

	let classes = $derived(
		type.includes('a2a') ? 'bg-accent-teal text-accent-teal-ink' :
		type.includes('mcp') ? 'bg-brand-200 text-brand-600' :
		type.includes('agentskill') ? 'bg-[#fdeccc] text-[#8a5a06]' :
		'bg-surface-tag text-ink'
	);
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold {classes}">
	{label}
</span>
