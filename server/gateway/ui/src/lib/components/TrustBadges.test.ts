// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render } from '@testing-library/svelte';
import { TRUST_STATUS_METADATA_KEY } from '../types';
import type { CatalogEntry } from '../types';
import TrustBadges from './TrustBadges.svelte';

afterEach(cleanup);

const bareEntry: CatalogEntry = {
	identifier: 'cid',
	displayName: 'Entry',
	type: 'application/ai-catalog+json'
};

function titles(container: HTMLElement): string[] {
	return [...container.querySelectorAll('[title]')].map((el) => el.getAttribute('title') ?? '');
}

describe('TrustBadges', () => {
	it('shows the trusted, verified, identity and scan badges a fully attested record earns', () => {
		const aicard: CatalogEntry = {
			...bareEntry,
			metadata: {
				[TRUST_STATUS_METADATA_KEY]: {
					trusted: true,
					verified: true,
					identityVerified: true,
					claims: [{ role: 'identity', subject: 'did:web:example.com', status: 'verified' }]
				},
				'agntcy.dir.security.v1.ScanResult': {
					isSafe: true,
					maxSeverity: 'NONE',
					reports: [{ scannerType: 'LLM', isSafe: true, maxSeverity: 'NONE' }]
				}
			}
		};

		const { container } = render(TrustBadges, { props: { aicard } });

		expect(titles(container)).toEqual([
			'Trusted — signature verification passed',
			'Verified — name ownership confirmed',
			'Identity verified — did:web:example.com',
			'Security Scanned — No issues found'
		]);
	});

	it('shows nothing for a record without trust metadata', () => {
		const { container } = render(TrustBadges, { props: { aicard: bareEntry } });

		expect(titles(container)).toEqual([]);
		expect(container.textContent?.trim()).toBe('');
	});
});
