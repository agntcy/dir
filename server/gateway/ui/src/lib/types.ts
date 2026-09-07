export interface AICardFilterCriteria {
	searchQuery: string;
	mediaTypes: Set<string>;
	statusFilters: Set<string>; // 'trusted' | 'verified' — independently selectable
	activeTags: Set<string>;
	scanSafe: boolean; // true = show only records where all scanners report safe
}

export interface ScanReportSummary {
	scannerType: string;
	isSafe: boolean;
	maxSeverity: string;
	updatedAt?: string;
}

export interface ScanManifest {
	isSafe: boolean;
	maxSeverity: string;
	reports: ScanReportSummary[];
}

export interface UsageMetrics {
	pullCount: number;
	lookupCount: number;
	providerCount: number;
}

export interface TrustStatus {
	trusted: boolean;
	verified: boolean;
}

export const TRUST_STATUS_METADATA_KEY = 'agntcy.dir.trust.v1.Status';

export interface CatalogTag {
	id: string;
	label: string;
}

/** One OASF class the extractor matched, with the score and tier that ranked it. */
export interface ScoredClass {
	id: number;
	/** Hierarchical OASF name, e.g. "natural_language_processing/text_classification". */
	name: string;
	/** Combined relevance score in [0,1]; higher is closer. */
	score: number;
	/** 1-based score group; tier 1 is the closest cluster of matches. */
	tier: number;
}

/**
 * Response of POST /v1/extract. Lists are absent rather than empty when the
 * extractor matched nothing, since protojson omits empty repeated fields.
 */
export interface ExtractTaxonomyResponse {
	skills?: ScoredClass[];
	domains?: ScoredClass[];
	/**
	 * Salient terms lifted from the text. They have no OASF class behind them and
	 * therefore no catalog tag, so tag-oriented consumers must ignore them.
	 */
	keywords?: string[];
}

/**
 * An extracted OASF class that some record in this registry actually carries,
 * i.e. one that survived the intersect against /v1/tags.
 */
export interface SuggestedTag {
	/** Catalog tag id, ready to hand to the same filter path as any other tag. */
	id: string;
	/** Display label, taken from the catalog tag so it reads like the rest of the list. */
	label: string;
	/** Full hierarchical OASF name, which the leaf-only label drops. */
	name: string;
	kind: 'skill' | 'domain';
	score: number;
}

export interface CatalogEntry {
	identifier: string;
	displayName: string;
	mediaType: string;
	data?: EntryData;
	version?: string;
	description?: string;
	tags?: string[];
	updatedAt?: string;
	trustManifest?: TrustManifest;
	metadata?: Record<string, unknown>;
}

export interface EntryData {
	entries?: SubEntry[];
	skillManifest?: SkillManifest;
	specVersion?: string;
}

export interface SubEntry {
	identifier?: string;
	displayName?: string;
	mediaType?: string;
	version?: string;
	data?: Record<string, unknown>;
}

export interface SkillManifest {
	name?: string;
	version?: string;
	description?: string;
}

export interface TrustManifest {
	identity?: string;
	identityType?: string;
	signature?: string;
	attestations?: unknown[];
	provenance?: unknown[];
}

export interface ExportFormat {
	format: string;
	label: string;
	ext: string;
}
