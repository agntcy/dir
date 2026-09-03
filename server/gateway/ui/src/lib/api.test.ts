// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	EXTRACT_TEXT_MAX_LEN,
	ExtractorUnavailableError,
	extractTaxonomy,
	suggestTagsFromExtraction
} from './api';
import type { CatalogTag, ExtractTaxonomyResponse } from './types';

/** Mirrors the `oasf:<schema>:skills:<name>` ids ListCatalogTags emits. */
function skillTag(name: string, schemaVersion = '*'): CatalogTag {
	return { id: `oasf:${schemaVersion}:skills:${name}`, label: leafLabel(name) };
}

function domainTag(name: string, schemaVersion = '*'): CatalogTag {
	return { id: `oasf:${schemaVersion}:domains:${name}`, label: leafLabel(name) };
}

function leafLabel(name: string): string {
	const leaf = name.slice(name.lastIndexOf('/') + 1);
	return leaf
		.split('_')
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

function scored(name: string, score: number, id = 1) {
	return { id, name, score, tier: 1 };
}

describe('suggestTagsFromExtraction', () => {
	it('maps extracted skills and domains onto the catalog tags carrying them', () => {
		const tags = [
			skillTag('natural_language_processing/text_classification'),
			domainTag('customer_support')
		];
		const extraction: ExtractTaxonomyResponse = {
			skills: [scored('natural_language_processing/text_classification', 0.9, 10304)],
			domains: [scored('customer_support', 0.7, 601)]
		};

		expect(suggestTagsFromExtraction(extraction, tags)).toEqual([
			{
				id: 'oasf:*:skills:natural_language_processing/text_classification',
				label: 'Text Classification',
				name: 'natural_language_processing/text_classification',
				kind: 'skill',
				score: 0.9
			},
			{
				id: 'oasf:*:domains:customer_support',
				label: 'Customer Support',
				name: 'customer_support',
				kind: 'domain',
				score: 0.7
			}
		]);
	});

	it('drops classes that no catalog tag carries', () => {
		const tags = [skillTag('natural_language_processing/text_classification')];
		const extraction: ExtractTaxonomyResponse = {
			skills: [
				scored('natural_language_processing/text_classification', 0.9),
				scored('audio/speech_to_text', 0.8)
			]
		};

		expect(suggestTagsFromExtraction(extraction, tags).map((s) => s.name)).toEqual([
			'natural_language_processing/text_classification'
		]);
	});

	it('matches on kind and class name regardless of the schema version in the tag id', () => {
		const tags = [skillTag('natural_language_processing/text_classification', '1.1.0')];
		const extraction: ExtractTaxonomyResponse = {
			skills: [scored('natural_language_processing/text_classification', 0.9)]
		};

		expect(suggestTagsFromExtraction(extraction, tags).map((s) => s.id)).toEqual([
			'oasf:1.1.0:skills:natural_language_processing/text_classification'
		]);
	});

	it('does not match a skill against a domain tag of the same name', () => {
		const tags = [domainTag('customer_support')];
		const extraction: ExtractTaxonomyResponse = {
			skills: [scored('customer_support', 0.9)]
		};

		expect(suggestTagsFromExtraction(extraction, tags)).toEqual([]);
	});

	it('preserves the extractor order: skills before domains, descending score within each', () => {
		const tags = [skillTag('a'), skillTag('b'), domainTag('c'), domainTag('d')];
		const extraction: ExtractTaxonomyResponse = {
			skills: [scored('b', 0.9), scored('a', 0.4)],
			domains: [scored('d', 0.8), scored('c', 0.3)]
		};

		expect(suggestTagsFromExtraction(extraction, tags).map((s) => s.name)).toEqual([
			'b',
			'a',
			'd',
			'c'
		]);
	});

	it('emits a tag at most once when the extractor repeats a class', () => {
		const tags = [skillTag('a')];
		const extraction: ExtractTaxonomyResponse = {
			skills: [scored('a', 0.9), scored('a', 0.5)]
		};

		expect(suggestTagsFromExtraction(extraction, tags).map((s) => s.score)).toEqual([0.9]);
	});

	it('ignores keywords, which have no OASF class behind them', () => {
		const tags = [skillTag('tickets'), { id: 'tickets', label: 'tickets' }];
		const extraction: ExtractTaxonomyResponse = { keywords: ['tickets'] };

		expect(suggestTagsFromExtraction(extraction, tags)).toEqual([]);
	});

	it('returns nothing when the extraction is empty or the catalog has no tags', () => {
		expect(suggestTagsFromExtraction({}, [skillTag('a')])).toEqual([]);
		expect(suggestTagsFromExtraction({ skills: [scored('a', 0.9)] }, [])).toEqual([]);
	});

	it('ignores non-OASF tag ids such as record annotations', () => {
		const tags = [{ id: 'team=platform', label: 'platform' }];
		const extraction: ExtractTaxonomyResponse = { skills: [scored('team=platform', 0.9)] };

		expect(suggestTagsFromExtraction(extraction, tags)).toEqual([]);
	});
});

describe('extractTaxonomy', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	function stubFetch(impl: (url: string, init?: RequestInit) => Response) {
		const fetchMock = vi.fn((url: string, init?: RequestInit) =>
			Promise.resolve(impl(url, init))
		);
		vi.stubGlobal('fetch', fetchMock);
		return fetchMock;
	}

	function jsonResponse(body: unknown, status = 200): Response {
		return new Response(JSON.stringify(body), {
			status,
			headers: { 'content-type': 'application/json' }
		});
	}

	it('posts the text to /v1/extract and returns the skills and domains', async () => {
		const fetchMock = stubFetch(() =>
			jsonResponse({
				skills: [scored('natural_language_processing/text_classification', 0.9)],
				keywords: ['tickets']
			})
		);

		const result = await extractTaxonomy('analyzing customer support tickets');

		expect(fetchMock).toHaveBeenCalledOnce();
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('/v1/extract');
		expect(init?.method).toBe('POST');
		expect(JSON.parse(String(init?.body))).toEqual({
			text: 'analyzing customer support tickets'
		});
		expect(result.skills?.[0].name).toBe(
			'natural_language_processing/text_classification'
		);
	});

	it('raises ExtractorUnavailableError when no extractor is configured', async () => {
		stubFetch(() => jsonResponse({ message: 'no OASF extractor is configured' }, 503));

		await expect(extractTaxonomy('anything')).rejects.toBeInstanceOf(
			ExtractorUnavailableError
		);
	});

	it('raises a plain error for other failures, so the action stays available', async () => {
		stubFetch(() => jsonResponse({ message: 'boom' }, 500));

		const err = await extractTaxonomy('anything').catch((e) => e);
		expect(err).toBeInstanceOf(Error);
		expect(err).not.toBeInstanceOf(ExtractorUnavailableError);
	});

	it('rejects text past the server limit without spending a request', async () => {
		const fetchMock = stubFetch(() => jsonResponse({}));

		await expect(extractTaxonomy('x'.repeat(EXTRACT_TEXT_MAX_LEN + 1))).rejects.toThrow();
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('rejects blank text without spending a request', async () => {
		const fetchMock = stubFetch(() => jsonResponse({}));

		await expect(extractTaxonomy('   ')).rejects.toThrow();
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('sends the trimmed text', async () => {
		const fetchMock = stubFetch(() => jsonResponse({}));

		await extractTaxonomy('  customer support  ');

		expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({
			text: 'customer support'
		});
	});
});
