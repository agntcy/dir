import type { AgentFilterCriteria, CatalogEntry } from './types';

const API_PAGE_SIZE = 100;

export function buildAgentFilterQuery(criteria: AgentFilterCriteria): string {
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

export async function fetchAgents(options: {
	filter?: string;
	pageSize?: number;
} = {}): Promise<CatalogEntry[]> {
	const pageSize = options.pageSize ?? API_PAGE_SIZE;
	let results: CatalogEntry[] = [];
	let pageToken = '';

	do {
		const query = new URLSearchParams();
		query.set('page_size', String(pageSize));
		if (options.filter) query.set('filter', options.filter);
		if (pageToken) query.set('page_token', pageToken);

		const resp = await fetch(`/v1/agents?${query}`);
		if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
		const data = await resp.json();
		results = results.concat(data.results || []);
		pageToken = data.nextPageToken || '';
	} while (pageToken);

	return results;
}
