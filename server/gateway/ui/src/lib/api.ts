import type {
	AICardFilterCriteria,
	CatalogEntry,
	CatalogTag,
	ExtractTaxonomyResponse,
	ScoredClass,
	SuggestedTag
} from './types';

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

/**
 * Mirrors the max_len on ExtractTaxonomyRequest.text. Counted in code points,
 * as the server counts runes.
 */
export const EXTRACT_TEXT_MAX_LEN = 1024;

/**
 * Raised when /v1/extract answers 503, which the gateway returns only when no
 * OASF extractor is configured or the backend is down. Distinct from a plain
 * error because the caller's response is to withdraw the action rather than to
 * report a failure the user can do anything about.
 */
export class ExtractorUnavailableError extends Error {
	constructor(message = 'taxonomy extraction is unavailable on this gateway') {
		super(message);
		this.name = 'ExtractorUnavailableError';
	}
}

/**
 * Maps free-form text onto the OASF taxonomy via POST /v1/extract.
 *
 * The returned classes are not checked against what any record here carries —
 * that is what suggestTagsFromExtraction is for.
 */
export async function extractTaxonomy(
	text: string,
	signal?: AbortSignal
): Promise<ExtractTaxonomyResponse> {
	const trimmed = text.trim();
	if (!trimmed) {
		throw new Error('text is required');
	}

	// Spend no request on text the server will reject anyway. Code points, not
	// `.length`: UTF-16 code units would over-count anything outside the BMP and
	// reject text the server would have accepted.
	const length = [...trimmed].length;
	if (length > EXTRACT_TEXT_MAX_LEN) {
		throw new Error(`text too long (${length} > ${EXTRACT_TEXT_MAX_LEN} characters)`);
	}

	const resp = await fetch('/v1/extract', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ text: trimmed }),
		signal
	});

	if (resp.status === 503) {
		throw new ExtractorUnavailableError();
	}

	if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);

	return await resp.json();
}

/** Kinds of OASF class that /v1/tags exposes as a tag. */
type ClassKind = SuggestedTag['kind'];

/**
 * Captures the kind and class name out of an `oasf:<schema>:skills|domains:<name>`
 * tag id. Ids of any other shape — record annotations, say — do not match.
 */
const OASF_TAG_ID = /^oasf:[^:]*:(skills|domains):(.+)$/;

/**
 * Indexes catalog tags by the OASF class they carry.
 *
 * Keyed on kind and class name rather than the whole tag id: ListCatalogTags
 * writes "*" as the schema version today, and hard-coding that would make every
 * suggestion silently vanish the day it emits a real version instead.
 */
function indexOASFTags(catalogTags: CatalogTag[]): Map<string, CatalogTag> {
	const byClass = new Map<string, CatalogTag>();

	for (const tag of catalogTags) {
		const match = OASF_TAG_ID.exec(tag.id);
		if (!match) continue;

		const kind: ClassKind = match[1] === 'skills' ? 'skill' : 'domain';
		const key = `${kind}:${match[2]}`;
		if (!byClass.has(key)) byClass.set(key, tag);
	}

	return byClass;
}

/**
 * Turns an extraction into selectable tags, keeping only the classes some record
 * in this registry actually carries.
 *
 * A suggestion no record has is worse than no suggestion: the user takes our
 * advice and lands on an empty catalog. Keywords are dropped here rather than in
 * the UI — they have no OASF class behind them, so there is no tag to select.
 *
 * Order follows the extractor: skills before domains, descending score within
 * each. Ties are not re-sorted, so equally scored classes keep the order the
 * extractor ranked them in.
 */
export function suggestTagsFromExtraction(
	extraction: ExtractTaxonomyResponse,
	catalogTags: CatalogTag[]
): SuggestedTag[] {
	const byClass = indexOASFTags(catalogTags);
	if (byClass.size === 0) return [];

	const suggestions: SuggestedTag[] = [];
	const seen = new Set<string>();

	const collect = (classes: ScoredClass[] | undefined, kind: ClassKind) => {
		for (const cl of classes ?? []) {
			const tag = byClass.get(`${kind}:${cl.name}`);
			if (!tag || seen.has(tag.id)) continue;

			seen.add(tag.id);
			suggestions.push({
				id: tag.id,
				label: tag.label,
				name: cl.name,
				kind,
				score: cl.score
			});
		}
	};

	collect(extraction.skills, 'skill');
	collect(extraction.domains, 'domain');

	return suggestions;
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
