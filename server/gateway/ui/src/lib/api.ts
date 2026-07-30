import type { AICardFilterCriteria, CatalogEntry, CatalogTag } from './types';

/** Matches the 3-column grid layout (18 = 6 full rows). */
export const CATALOG_PAGE_SIZE = 18;

export interface AICardsPage {
	results: CatalogEntry[];
	nextPageToken: string;
	totalCount: number;
}

/** Returns the decimal offset expected by ListAgents page_token. */
export function pageTokenForPage(page: number, pageSize = CATALOG_PAGE_SIZE): string {
	const offset = (page - 1) * pageSize;
	return offset > 0 ? String(offset) : '';
}

function formatFilterToken(value: string): string {
	if (/[",=]/.test(value) || value.includes(',')) {
		return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
	}

	return value;
}

export function buildAICardFilterQuery(criteria: AICardFilterCriteria): string {
	const clauses: string[] = [];

	const search = criteria.searchQuery.trim();
	if (search) {
		clauses.push(`displayName=${formatFilterToken(search)}`);
	}

	if (!criteria.mediaTypes.has('all')) {
		const types = [...criteria.mediaTypes].filter((t) => t !== 'all');
		if (types.length > 0) {
			clauses.push(`type=${types.join(',')}`);
		}
	}

	if (criteria.statusFilters.has('verified')) {
		clauses.push('verified=true');
	}

	if (criteria.statusFilters.has('trusted')) {
		clauses.push('trusted=true');
	}

	if (criteria.scanSafe) {
		clauses.push('safe=true');
	}

	if (criteria.activeTags.size > 0) {
		const tags = [...criteria.activeTags].map(formatFilterToken).join(',');
		clauses.push(`tags=${tags}`);
	}

	return clauses.join(' AND ');
}

export async function fetchCatalogTags(signal?: AbortSignal): Promise<CatalogTag[]> {
	const resp = await fetch('/v1/tags', { signal });
	if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);

	const data = await resp.json();
	return data.tags || [];
}

export async function fetchAICardsPage(
	options: {
		filter?: string;
		pageSize?: number;
		pageToken?: string;
		signal?: AbortSignal;
	} = {}
): Promise<AICardsPage> {
	const pageSize = options.pageSize ?? CATALOG_PAGE_SIZE;
	const query = new URLSearchParams();
	query.set('page_size', String(pageSize));
	if (options.filter) query.set('filter', options.filter);
	if (options.pageToken) query.set('page_token', options.pageToken);

	const resp = await fetch(`/v1/agents?${query}`, { signal: options.signal });
	if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);

	const data = await resp.json();
	return {
		results: data.results || [],
		nextPageToken: data.nextPageToken || '',
		totalCount: data.totalCount ?? 0,
	};
}
