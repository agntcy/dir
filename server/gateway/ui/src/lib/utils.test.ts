// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { TRUST_STATUS_METADATA_KEY } from './types';
import type { CatalogEntry, ClaimSummary } from './types';
import {
	extractShortTag,
	formatCheckedAt,
	getClaims,
	getIdentityClaim,
	getTrustStatus,
	isStatusFilter
} from './utils';

const bareEntry: CatalogEntry = {
	identifier: 'cid',
	displayName: 'Entry',
	type: 'application/ai-catalog+json'
};

function entryWithTrustStatus(status: unknown): CatalogEntry {
	return { ...bareEntry, metadata: { [TRUST_STATUS_METADATA_KEY]: status } };
}

const identityClaim: ClaimSummary = {
	role: 'identity',
	subject: 'did:web:example.com',
	status: 'failed',
	verifiedAt: '2026-10-12T19:48:08Z'
};

const ownerClaim: ClaimSummary = { role: 'owner', subject: 'dns:example.com', status: 'verified' };

describe('extractShortTag', () => {
	it.each<[string, string]>([
		['oasf:1.0.0:skills:natural_language_processing/text_completion', 'Text Completion'],
		['oasf:1.0.0:skills:test_skill', 'Test Skill'],
		['oasf:*:domains:customer_support', 'Customer Support'],
		['agntcy.dir/identity=ans://v1.0.0.host', 'agntcy.dir/identity=ans://v1.0.0.host'],
		['owner=alice', 'owner=alice'],
		['featured', 'featured'],
		['team:platform_eng', 'team:platform_eng']
	])('renders %s as %s', (tag, expected) => {
		expect(extractShortTag(tag)).toBe(expected);
	});
});

describe('getTrustStatus', () => {
	it('passes identityVerified and claims through', () => {
		const status = {
			trusted: true,
			verified: false,
			identityVerified: true,
			claims: [identityClaim, ownerClaim]
		};

		expect(getTrustStatus(entryWithTrustStatus(status))).toEqual(status);
	});

	it('returns null when the entry has no metadata', () => {
		expect(getTrustStatus(bareEntry)).toBeNull();
	});

	it.each<[string, unknown]>([
		['the key is absent', undefined],
		['the status is not an object', 'verified'],
		['trusted is missing', { verified: true }],
		['verified is missing', { trusted: true }],
		['a flag is not a boolean', { trusted: 'true', verified: true }]
	])('returns null when %s', (_name, status) => {
		expect(getTrustStatus(entryWithTrustStatus(status))).toBeNull();
	});

	it('drops identityVerified when it is not a boolean', () => {
		const entry = entryWithTrustStatus({ trusted: true, verified: false, identityVerified: 'true' });

		expect(getTrustStatus(entry)).toEqual({ trusted: true, verified: false });
	});

	it.each<[string, unknown]>([
		['the status does not list claims', { trusted: true, verified: true }],
		['claims is not a list', { trusted: true, verified: true, claims: 'none' }]
	])('leaves claims out when %s', (_name, status) => {
		expect(getTrustStatus(entryWithTrustStatus(status))).toEqual({ trusted: true, verified: true });
	});

	it('drops claim elements that do not match the claim shape', () => {
		const claims = [
			identityClaim,
			{ role: 'owner', subject: 'dns:example.com' },
			{ role: 'owner', subject: 'dns:example.com', status: 'verified', verifiedAt: 5 },
			{ role: 1, subject: 'dns:example.com', status: 'verified' },
			'owner',
			null
		];

		expect(getTrustStatus(entryWithTrustStatus({ trusted: false, verified: false, claims }))).toEqual({
			trusted: false,
			verified: false,
			claims: [identityClaim]
		});
	});

	it('keeps the first claim seen for each role', () => {
		const laterIdentityClaim = { ...identityClaim, subject: 'did:web:other.example', status: 'verified' };
		const entry = entryWithTrustStatus({
			trusted: false,
			verified: false,
			claims: [identityClaim, laterIdentityClaim, ownerClaim]
		});

		expect(getTrustStatus(entry)?.claims).toEqual([identityClaim, ownerClaim]);
	});
});

describe('getClaims', () => {
	it('returns the validated claims', () => {
		const entry = entryWithTrustStatus({
			trusted: false,
			verified: false,
			claims: [identityClaim, ownerClaim]
		});

		expect(getClaims(entry)).toEqual([identityClaim, ownerClaim]);
	});

	it.each<[string, CatalogEntry]>([
		['the entry has no metadata', bareEntry],
		['the status has no claims', entryWithTrustStatus({ trusted: true, verified: true })],
		['claims is not a list', entryWithTrustStatus({ trusted: true, verified: true, claims: 'none' })],
		['the status is not an object', entryWithTrustStatus('verified')],
		['the status lacks its booleans', entryWithTrustStatus({ claims: [identityClaim] })]
	])('returns an empty list when %s', (_name, entry) => {
		expect(getClaims(entry)).toEqual([]);
	});
});

describe('getIdentityClaim', () => {
	it('selects the identity claim by role', () => {
		const entry = entryWithTrustStatus({
			trusted: false,
			verified: true,
			claims: [ownerClaim, identityClaim]
		});

		expect(getIdentityClaim(entry)).toEqual(identityClaim);
	});

	it('returns null when only other roles are claimed', () => {
		const entry = entryWithTrustStatus({ trusted: false, verified: true, claims: [ownerClaim] });

		expect(getIdentityClaim(entry)).toBeNull();
	});

	it('returns null when the record has no claims', () => {
		expect(getIdentityClaim(entryWithTrustStatus({ trusted: false, verified: false }))).toBeNull();
	});
});

describe('formatCheckedAt', () => {
	it('localizes a parseable time', () => {
		const verifiedAt = '2026-10-12T19:48:08Z';

		expect(formatCheckedAt(verifiedAt)).toBe(new Date(verifiedAt).toLocaleString());
	});

	it.each<[string, string | undefined]>([
		['absent', undefined],
		['empty', ''],
		['unparseable', 'yesterday']
	])('returns nothing when the time is %s', (_name, verifiedAt) => {
		expect(formatCheckedAt(verifiedAt)).toBe('');
	});
});

describe('isStatusFilter', () => {
	it.each<[string, boolean]>([
		['trusted', true],
		['identityVerified', true],
		['verified', false],
		['', false]
	])('classifies %j as %s', (value, expected) => {
		expect(isStatusFilter(value)).toBe(expected);
	});
});
