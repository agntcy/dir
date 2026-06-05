<script lang="ts">
	interface Props {
		type: string;
	}

	let { type }: Props = $props();

	let label = $derived(
		type.includes('a2a') ? 'A2A' :
		type.includes('mcp') ? 'MCP' :
		type.includes('ai-skill') ? 'SKILL' :
		type.split('/').pop()?.split('+')[0] || type
	);

	let classes = $derived(
		type.includes('a2a') ? 'bg-purple-100 text-purple-800' :
		type.includes('mcp') ? 'bg-green-100 text-green-800' :
		type.includes('ai-skill') ? 'bg-amber-100 text-amber-800' :
		'bg-gray-100 text-gray-800'
	);
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {classes}">
	{label}
</span>
