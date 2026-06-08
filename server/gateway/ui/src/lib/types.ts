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
