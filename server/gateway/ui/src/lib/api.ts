import type { AgentFilterCriteria, CatalogEntry } from './types';

/** Matches the 3-column grid layout (18 = 6 full rows). */
export const CATALOG_PAGE_SIZE = 18;

export interface AgentsPage {
	results: CatalogEntry[];
	nextPageToken: string;
}

export function buildAgentFilterQuery(
	criteria: Pick<AgentFilterCriteria, 'searchQuery' | 'mediaTypes'>
): string {
	const clauses: string[] = [];

	const search = criteria.searchQuery.trim();
	if (search) {
		clauses.push(`displayName=${search}`);
	}

	if (!criteria.mediaTypes.has('all')) {
		const types = [...criteria.mediaTypes].filter((t) => t !== 'all');
		if (types.length > 0) {
			clauses.push(`type=${types.join(',')}`);
		}
	}

	return clauses.join(' AND ');
}

export async function fetchAgentsPage(
	options: {
		filter?: string;
		pageSize?: number;
		pageToken?: string;
		signal?: AbortSignal;
	} = {}
): Promise<AgentsPage> {
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
		nextPageToken: data.nextPageToken || ''
	};
}
