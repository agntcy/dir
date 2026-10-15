// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/svelte';
import ClaimsSection from './ClaimsSection.svelte';

afterEach(cleanup);

describe('ClaimsSection', () => {
	it('lists each claim with its role, outcome, subject and check time', () => {
		const verifiedAt = '2026-10-12T19:48:08Z';
		render(ClaimsSection, {
			props: {
				claims: [
					{ role: 'identity', subject: 'did:web:example.com', status: 'failed', verifiedAt },
					{ role: 'owner', subject: 'dns:example.com', status: 'verified' }
				]
			}
		});

		expect(screen.getByText('Claims')).toBeTruthy();
		expect(screen.getByText('Identity claim')).toBeTruthy();
		expect(screen.getByText('Owner claim')).toBeTruthy();
		expect(screen.getByText('did:web:example.com')).toBeTruthy();
		expect(screen.getByText('dns:example.com')).toBeTruthy();
		expect(screen.getByText(new Date(verifiedAt).toLocaleString())).toBeTruthy();

		const failed = screen.getByText('Failed');
		const verified = screen.getByText('Verified');
		expect(failed.className).toContain('text-red-700');
		expect(verified.className).toContain('text-emerald-700');
	});

	it('shows a status it does not know verbatim', () => {
		render(ClaimsSection, {
			props: { claims: [{ role: 'identity', subject: 'did:web:example.com', status: 'pending' }] }
		});

		expect(screen.getByText('pending')).toBeTruthy();
	});

	it('leaves the check time out when it does not parse', () => {
		render(ClaimsSection, {
			props: {
				claims: [
					{ role: 'identity', subject: 'did:web:example.com', status: 'verified', verifiedAt: 'yesterday' }
				]
			}
		});

		expect(screen.queryByText('Invalid Date')).toBeNull();
		const header = screen.getByText('Identity claim').parentElement;
		expect(header?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Identity claim Verified');
	});

	it('renders nothing for an empty list', () => {
		const { container } = render(ClaimsSection, { props: { claims: [] } });

		expect(container.querySelector('div')).toBeNull();
		expect(container.textContent?.trim()).toBe('');
	});
});
