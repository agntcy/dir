// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render } from '@testing-library/svelte';
import IdentityBadge from './IdentityBadge.svelte';

afterEach(cleanup);

function renderBadge(props: { status: string; subject: string; verifiedAt?: string }) {
	const { container } = render(IdentityBadge, { props });

	return container.querySelector('[title]');
}

describe('IdentityBadge', () => {
	it('shows a verified claim in green, titled with the subject', () => {
		const badge = renderBadge({ status: 'verified', subject: 'did:web:example.com' });

		expect(badge?.getAttribute('title')).toBe('Identity verified — did:web:example.com');
		expect(badge?.querySelector('svg')?.getAttribute('class')).toContain('text-emerald-600');
	});

	it('shows a failed claim in red', () => {
		const badge = renderBadge({ status: 'failed', subject: 'did:web:example.com' });

		expect(badge?.getAttribute('title')).toBe('Identity check failed — did:web:example.com');
		expect(badge?.querySelector('svg')?.getAttribute('class')).toContain('text-red-600');
	});

	it('appends the check time when verifiedAt parses', () => {
		const verifiedAt = '2026-10-12T19:48:08Z';
		const badge = renderBadge({ status: 'verified', subject: 'did:web:example.com', verifiedAt });

		expect(badge?.getAttribute('title')).toBe(
			`Identity verified — did:web:example.com (checked ${new Date(verifiedAt).toLocaleString()})`
		);
	});

	it('omits the check time when verifiedAt does not parse', () => {
		const badge = renderBadge({
			status: 'failed',
			subject: 'did:web:example.com',
			verifiedAt: 'yesterday'
		});

		expect(badge?.getAttribute('title')).toBe('Identity check failed — did:web:example.com');
	});

	it('keeps a long subject whole', () => {
		const subject = `did:web:${'a'.repeat(300)}`;
		const badge = renderBadge({ status: 'verified', subject });

		expect(badge?.getAttribute('title')).toContain(subject);
	});

	it('renders nothing for a status it does not know', () => {
		const { container } = render(IdentityBadge, {
			props: { status: 'pending', subject: 'did:web:example.com' }
		});

		expect(container.querySelector('[title]')).toBeNull();
		expect(container.querySelector('svg')).toBeNull();
	});
});
