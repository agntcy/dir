import type { CatalogEntry, ClaimSummary, ScanManifest, StatusFilter, SubEntry, ExportFormat, UsageMetrics, TrustStatus } from './types';
import { STATUS_FILTERS, TRUST_STATUS_METADATA_KEY } from './types';

export function getScanManifest(aicard: CatalogEntry): ScanManifest | null {
	const sm = aicard.metadata?.["agntcy.dir.security.v1.ScanResult"] as ScanManifest | undefined;
	if (!sm || !Array.isArray(sm.reports) || sm.reports.length === 0) return null;
	return sm;
}

export function hasScanManifest(aicard: CatalogEntry): boolean {
	return getScanManifest(aicard) !== null;
}

export function extractEntryTypes(aicard: CatalogEntry): string[] {
	const entries = aicard.data?.entries || [];
	if (entries.length > 0) return entries.map((e) => e.type || '');
	if (aicard.type && aicard.type !== 'application/ai-catalog+json')
		return [aicard.type];
	return [];
}

/**
 * Captures the kind and class name out of an `oasf:<schema>:skills|domains:<name>`
 * tag id. Ids of any other shape — record annotations, say — do not match.
 */
export const OASF_TAG_ID = /^oasf:[^:]*:(skills|domains):(.+)$/;

/**
 * Only OASF class tags are shortened. Annotation tags are `key=value`, and a
 * URI value would not survive being split on `:` or `/`.
 */
export function extractShortTag(tag: string): string {
	const match = OASF_TAG_ID.exec(tag);
	if (!match) return tag;

	const leaf = match[2].slice(match[2].lastIndexOf('/') + 1);
	return leaf
		.split('_')
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(' ');
}

function isClaimSummary(value: unknown): value is ClaimSummary {
	if (typeof value !== 'object' || value === null) return false;
	const claim = value as Record<string, unknown>;
	return (
		typeof claim.role === 'string' &&
		typeof claim.subject === 'string' &&
		typeof claim.status === 'string' &&
		(claim.verifiedAt === undefined || typeof claim.verifiedAt === 'string')
	);
}

function firstClaimPerRole(claims: ClaimSummary[]): ClaimSummary[] {
	const byRole = new Map<string, ClaimSummary>();
	for (const claim of claims) {
		if (!byRole.has(claim.role)) byRole.set(claim.role, claim);
	}
	return [...byRole.values()];
}

/** The one validated parse of the trust metadata; badges and the claims list both read from it. */
export function getTrustStatus(aicard: CatalogEntry): TrustStatus | null {
	const raw = aicard.metadata?.[TRUST_STATUS_METADATA_KEY];
	if (typeof raw !== 'object' || raw === null) return null;

	const status = raw as Record<string, unknown>;
	if (typeof status.trusted !== 'boolean' || typeof status.verified !== 'boolean') return null;

	const result: TrustStatus = { trusted: status.trusted, verified: status.verified };
	if (typeof status.identityVerified === 'boolean') result.identityVerified = status.identityVerified;
	if (Array.isArray(status.claims)) result.claims = firstClaimPerRole(status.claims.filter(isClaimSummary));

	return result;
}

export function getClaims(aicard: CatalogEntry): ClaimSummary[] {
	return getTrustStatus(aicard)?.claims ?? [];
}

export function getIdentityClaim(aicard: CatalogEntry): ClaimSummary | null {
	return getClaims(aicard).find((claim) => claim.role === 'identity') ?? null;
}

/** Localized time of a claim check, or '' when there is nothing parseable to show. */
export function formatCheckedAt(verifiedAt?: string): string {
	const time = verifiedAt ? new Date(verifiedAt) : null;
	return time && !Number.isNaN(time.getTime()) ? time.toLocaleString() : '';
}

export function isStatusFilter(value: string): value is StatusFilter {
	return STATUS_FILTERS.some((filter) => filter.value === value);
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
	const mt = entry.type || '';
	const data = entry.data as Record<string, unknown> | undefined;
	if (mt.includes('a2a')) {
		const card = data?.card_data as Record<string, unknown> | undefined;
		return (card?.name as string) || entry.displayName || 'Unnamed';
	}
	if (mt.includes('mcp')) return (data?.name as string) || entry.displayName || 'Unnamed';
	return entry.displayName || 'Unnamed';
}

export function extractEntryVersion(entry: SubEntry): string {
	const mt = entry.type || '';
	const data = entry.data as Record<string, unknown> | undefined;
	if (mt.includes('a2a')) {
		const card = data?.card_data as Record<string, unknown> | undefined;
		return (card?.version as string) || entry.version || '-';
	}
	return entry.version || '-';
}
