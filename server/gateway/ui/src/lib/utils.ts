import type { AICardFilterCriteria, CatalogEntry, SubEntry, ExportFormat } from './types';

export function hasActiveClientFilters(criteria: AICardFilterCriteria): boolean {
	return (
		criteria.activeTags.size > 0 ||
		criteria.statusFilter === 'trusted' ||
		criteria.statusFilter === 'verified'
	);
}

export function collectSortedTags(aicards: CatalogEntry[]): string[] {
	return Array.from(new Set(aicards.flatMap((aicard) => aicard.tags || []))).sort();
}

export function mergeSortedTags(existing: string[], newAicards: CatalogEntry[]): string[] {
	if (newAicards.length === 0) return existing;

	const seen = new Set(existing);
	let added = false;

	for (const aicard of newAicards) {
		for (const tag of aicard.tags || []) {
			if (!seen.has(tag)) {
				seen.add(tag);
				added = true;
			}
		}
	}

	return added ? Array.from(seen).sort() : existing;
}

export function applyClientFilters(
	aicards: CatalogEntry[],
	criteria: AICardFilterCriteria
): CatalogEntry[] {
	return aicards.filter((aicard) => {
		if (criteria.activeTags.size > 0) {
			const aicardTags = new Set(aicard.tags || []);
			let hasAny = false;
			for (const t of criteria.activeTags) {
				if (aicardTags.has(t)) {
					hasAny = true;
					break;
				}
			}
			if (!hasAny) return false;
		}

		if (criteria.statusFilter === 'trusted' || criteria.statusFilter === 'verified') {
			if (!hasTrustManifest(aicard)) return false;
		}

		return true;
	});
}

export function extractEntryTypes(aicard: CatalogEntry): string[] {
	const entries = aicard.data?.entries || [];
	if (entries.length > 0) return entries.map((e) => e.mediaType || '');
	if (aicard.mediaType && aicard.mediaType !== 'application/ai-catalog+json')
		return [aicard.mediaType];
	return [];
}

export function extractShortTag(tag: string): string {
	if (tag.startsWith('oasf:')) {
		const segment = tag.split('/').pop() || '';
		return segment
			.split('_')
			.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
			.join(' ');
	}
	const parts = tag.split(':');
	return (parts[parts.length - 1] || '').replace(/_/g, ' ');
}

export function hasTrustManifest(aicard: CatalogEntry): boolean {
	return !!(aicard.trustManifest && aicard.trustManifest.signature);
}

export function fakeStats(aicard: CatalogEntry) {
	let hash = 0;
	const id = aicard.identifier || aicard.displayName || '';
	for (let i = 0; i < id.length; i++) hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0;
	const seed = Math.abs(hash);
	return {
		downloads: 100 + (seed % 9900),
		rating: 3 + (seed % 20) / 10,
		providers: 1 + (seed % 12)
	};
}

export function formatDownloads(n: number): string {
	if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
	return n.toString();
}

export function extractCid(identifier: string): string {
	if (!identifier) return '';
	const parts = identifier.split(':');
	return parts[parts.length - 1];
}

export function exportFormatForType(mediaType: string): ExportFormat {
	if (mediaType.includes('a2a')) return { format: 'a2a', label: 'Download JSON', ext: 'json' };
	if (mediaType.includes('mcp'))
		return { format: 'mcp-ghcopilot', label: 'Download JSON', ext: 'json' };
	if (mediaType.includes('agentskill') && mediaType.endsWith('+md'))
		return { format: 'agent-skill', label: 'Download Markdown', ext: 'md' };
	if (mediaType.includes('agentskill') && mediaType.endsWith('+tar.gz'))
		return { format: 'agent-skill-bundle', label: 'Download Bundle', ext: 'tar.gz' };
	return { format: 'oasf', label: 'Download Asset', ext: 'json' };
}

export function extractEntryName(entry: SubEntry): string {
	const mt = entry.mediaType || '';
	const data = entry.data as Record<string, unknown> | undefined;
	if (mt.includes('a2a')) {
		const card = data?.card_data as Record<string, unknown> | undefined;
		return (card?.name as string) || entry.displayName || 'Unnamed';
	}
	if (mt.includes('mcp')) return (data?.name as string) || entry.displayName || 'Unnamed';
	return entry.displayName || 'Unnamed';
}

export function extractEntryVersion(entry: SubEntry): string {
	const mt = entry.mediaType || '';
	const data = entry.data as Record<string, unknown> | undefined;
	if (mt.includes('a2a')) {
		const card = data?.card_data as Record<string, unknown> | undefined;
		return (card?.version as string) || entry.version || '-';
	}
	return entry.version || '-';
}
