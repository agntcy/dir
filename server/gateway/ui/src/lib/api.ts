import type { CatalogEntry } from './types';

export async function fetchAllAgents(): Promise<CatalogEntry[]> {
	let results: CatalogEntry[] = [];
	let pageToken = '';

	do {
		const url = pageToken
			? `/v1/agents?page_token=${encodeURIComponent(pageToken)}`
			: '/v1/agents';
		const resp = await fetch(url);
		if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
		const data = await resp.json();
		results = results.concat(data.results || []);
		pageToken = data.nextPageToken || '';
	} while (pageToken);

	return results;
}
