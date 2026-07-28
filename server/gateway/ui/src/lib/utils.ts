import type { AICardFilterCriteria, CatalogEntry, ScanManifest, SubEntry, ExportFormat, UsageMetrics, TrustStatus } from './types';
import { TRUST_STATUS_METADATA_KEY } from './types';

export function hasActiveClientFilters(criteria: AICardFilterCriteria): boolean {
	return (
		criteria.activeTags.size > 0 ||
		criteria.statusFilters.size > 0 ||
		criteria.scanSafe
	);
}

export function getScanManifest(aicard: CatalogEntry): ScanManifest | null {
	const sm = aicard.metadata?.["agntcy.dir.security.v1.ScanResult"] as ScanManifest | undefined;
	if (!sm || !Array.isArray(sm.reports) || sm.reports.length === 0) return null;
	return sm;
}

export function hasScanManifest(aicard: CatalogEntry): boolean {
	return getScanManifest(aicard) !== null;
}

export function applyClientFilters(
	aicards: CatalogEntry[],
	criteria: AICardFilterCriteria
): CatalogEntry[] {
	return aicards.filter((aicard) => {
		if (criteria.activeTags.size > 0) {
			const aicardTags = new Set(aicard.tags || []);
			if (!entryMatchesAnyActiveTag(aicardTags, criteria.activeTags)) return false;
		}

		if (criteria.statusFilters.size > 0) {
			const trustStatus = getTrustStatus(aicard);
			for (const filter of criteria.statusFilters) {
				if (filter === 'trusted' && !trustStatus?.trusted) return false;
				if (filter === 'verified' && !trustStatus?.verified) return false;
			}
		}

		if (criteria.scanSafe) {
			const sm = getScanManifest(aicard);
			if (!sm || !sm.isSafe) return false;
		}

		return true;
	});
}

function entryMatchesAnyActiveTag(entryTags: Set<string>, activeTags: Set<string>): boolean {
	for (const filterTag of activeTags) {
		if (entryTags.has(filterTag)) return true;

		for (const entryTag of entryTags) {
			if (catalogTagMatchesFilter(filterTag, entryTag)) return true;
		}
	}

	return false;
}

export function catalogTagMatchesFilter(filterTag: string, entryTag: string): boolean {
	const filterParts = filterTag.split(':');
	const entryParts = entryTag.split(':');

	if (filterParts.length !== entryParts.length) return false;

	for (let i = 0; i < filterParts.length; i++) {
		if (filterParts[i] === '*') continue;
		if (filterParts[i] !== entryParts[i]) return false;
	}

	return true;
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

export function getTrustStatus(aicard: CatalogEntry): TrustStatus | null {
	const status = aicard.metadata?.[TRUST_STATUS_METADATA_KEY] as TrustStatus | undefined;
	if (!status || typeof status.trusted !== 'boolean' || typeof status.verified !== 'boolean') {
		return null;
	}

	return status;
}

export function getUsageMetrics(aicard: CatalogEntry): UsageMetrics | null {
	const m = aicard.metadata?.['agntcy.dir.usage.v1.Metrics'] as UsageMetrics | undefined;
	if (!m || typeof m.pullCount !== 'number') return null;
	return m;
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
	if (mediaType.includes('agent-skills') && mediaType.endsWith('+md'))
		return { format: 'agent-skill', label: 'Download Markdown', ext: 'md' };
	if (mediaType.includes('agent-skills') && mediaType.endsWith('+gzip'))
		return { format: 'agent-skill-bundle', label: 'Download Bundle', ext: 'gzip' };
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
