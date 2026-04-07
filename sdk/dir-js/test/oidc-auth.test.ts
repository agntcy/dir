// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { env } from 'node:process';

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { Client, Config, TokenCache, TOKEN_CACHE_FILE } from '../src/client/client.js';
import * as oauthPkce from '../src/client/oauthPkce.js';

describe('OIDC auth config', () => {
  const originalEnv = { ...env };

  afterEach(() => {
    for (const key of Object.keys(env)) {
      if (!(key in originalEnv)) {
        delete env[key];
      }
    }
    for (const [k, v] of Object.entries(originalEnv)) {
      env[k] = v;
    }
  });

  test('loadFromEnv uses AUTH_TOKEN', () => {
    env['DIRECTORY_CLIENT_AUTH_TOKEN'] = 'primary-token';
    const config = Config.loadFromEnv();
    expect(config.authToken).toBe('primary-token');
    expect(config.oidcAccessToken).toBe('primary-token');
  });

  test('loadFromEnv ignores legacy OIDC/OAUTH access token env names', () => {
    env['DIRECTORY_CLIENT_OIDC_ACCESS_TOKEN'] = 'legacy-token';
    env['DIRECTORY_CLIENT_OAUTH_ACCESS_TOKEN'] = 'older-legacy-token';
    const config = Config.loadFromEnv();
    expect(config.authToken).toBe('');
    expect(config.oidcAccessToken).toBe('');
  });

  test('Config has no machine-flow OIDC fields', () => {
    const config = new Config();
    expect('oidcMachineClientId' in config).toBe(false);
    expect('oidcMachineClientSecret' in config).toBe(false);
  });

  test('token cache uses XDG_CONFIG_HOME dirctl path', () => {
    const tmp = mkdtempSync(join(tmpdir(), 'dir-sdk-oidc-'));
    env['XDG_CONFIG_HOME'] = tmp;
    const cache = new TokenCache();
    expect(cache.getCachePath()).toBe(join(tmp, 'dirctl', TOKEN_CACHE_FILE));
  });
});

describe('OIDC auth client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test('constructor uses pre-issued token without calling discovery or login', () => {
    const fetchSpy = vi.spyOn(oauthPkce, 'fetchOpenidConfiguration');
    const loginSpy = vi.spyOn(oauthPkce, 'runLoopbackPkceLogin');
    const cacheSpy = vi.spyOn(TokenCache.prototype, 'getValidToken').mockReturnValue(undefined);

    const config = new Config(
      'directory.example.com:443',
      Config.DEFAULT_DIRCTL_PATH,
      Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
      'oidc',
      Config.DEFAULT_JWT_AUDIENCE,
      Config.DEFAULT_TLS_CA_FILE,
      Config.DEFAULT_TLS_CERT_FILE,
      Config.DEFAULT_TLS_KEY_FILE,
      'preissued-token',
    );
    const client = new Client(config);

    expect(client['oauthHolder']?.getAccessToken()).toBe('preissued-token');
    expect(fetchSpy).not.toHaveBeenCalled();
    expect(loginSpy).not.toHaveBeenCalled();
    cacheSpy.mockRestore();
  });

  test('constructor without token does not start PKCE', () => {
    const fetchSpy = vi.spyOn(oauthPkce, 'fetchOpenidConfiguration');
    const loginSpy = vi.spyOn(oauthPkce, 'runLoopbackPkceLogin');
    vi.spyOn(TokenCache.prototype, 'getValidToken').mockReturnValue(undefined);

    const config = new Config(
      'directory.example.com:443',
      Config.DEFAULT_DIRCTL_PATH,
      Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
      'oidc',
    );
    const client = new Client(config);

    expect(() => client['oauthHolder']?.getAccessToken()).toThrow(
      /DIRECTORY_CLIENT_AUTH_TOKEN/,
    );
    expect(fetchSpy).not.toHaveBeenCalled();
    expect(loginSpy).not.toHaveBeenCalled();
  });

  test('constructor uses cached token', () => {
    vi.spyOn(oauthPkce, 'fetchOpenidConfiguration');
    vi.spyOn(oauthPkce, 'runLoopbackPkceLogin');
    const tmp = mkdtempSync(join(tmpdir(), 'dir-sdk-oidc-cache-'));
    env['XDG_CONFIG_HOME'] = tmp;
    const cacheDir = join(tmp, 'dirctl');
    mkdirSync(cacheDir, { recursive: true });
    const expiresAt = new Date(Date.now() + 3_600_000).toISOString().replace(/\.\d{3}Z$/, 'Z');
    const createdAt = new Date().toISOString().replace(/\.\d{3}Z$/, 'Z');
    writeFileSync(
      join(cacheDir, TOKEN_CACHE_FILE),
      `${JSON.stringify(
        {
          access_token: 'cached-token',
          token_type: 'bearer',
          provider: 'oidc',
          issuer: 'https://issuer.example.com',
          refresh_token: 'cached-refresh-token',
          expires_at: expiresAt,
          created_at: createdAt,
        },
        null,
        2,
      )}\n`,
    );

    const config = new Config(
      'directory.example.com:443',
      Config.DEFAULT_DIRCTL_PATH,
      Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
      'oidc',
    );
    const client = new Client(config);

    expect(client['oauthHolder']?.getAccessToken()).toBe('cached-token');
  });

  test('authenticateOAuthPkce updates access token', async () => {
    const tmp = mkdtempSync(join(tmpdir(), 'dir-sdk-oidc-auth-'));
    env['XDG_CONFIG_HOME'] = tmp;

    vi.spyOn(oauthPkce, 'fetchOpenidConfiguration').mockResolvedValue({
      authorization_endpoint: 'https://issuer.example.com/auth',
      token_endpoint: 'https://issuer.example.com/token',
    });
    vi.spyOn(oauthPkce, 'runLoopbackPkceLogin').mockResolvedValue({
      access_token: 'fresh-token',
      refresh_token: 'ignored-refresh-token',
      expires_in: 3600,
    });

    const config = new Config(
      'directory.example.com:443',
      Config.DEFAULT_DIRCTL_PATH,
      Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
      'oidc',
      Config.DEFAULT_JWT_AUDIENCE,
      Config.DEFAULT_TLS_CA_FILE,
      Config.DEFAULT_TLS_CERT_FILE,
      Config.DEFAULT_TLS_KEY_FILE,
      '',
      Config.DEFAULT_TLS_SERVER_NAME,
      Config.DEFAULT_TLS_SKIP_VERIFY,
      'https://issuer.example.com',
      'client-id',
    );
    const client = new Client(config);
    await client.authenticateOAuthPkce();

    expect(client['oauthHolder']?.getAccessToken()).toBe('fresh-token');
    expect(oauthPkce.fetchOpenidConfiguration).toHaveBeenCalledOnce();
    expect(oauthPkce.runLoopbackPkceLogin).toHaveBeenCalledOnce();
  });

  test('authenticateOAuthPkce saves cache entry compatible with tooling', async () => {
    const tmp = mkdtempSync(join(tmpdir(), 'dir-sdk-oidc-save-'));
    env['XDG_CONFIG_HOME'] = tmp;

    vi.spyOn(oauthPkce, 'fetchOpenidConfiguration').mockResolvedValue({
      authorization_endpoint: 'https://issuer.example.com/auth',
      token_endpoint: 'https://issuer.example.com/token',
    });
    vi.spyOn(oauthPkce, 'runLoopbackPkceLogin').mockResolvedValue({
      access_token: 'fresh-token',
      refresh_token: 'refresh-token',
      token_type: 'bearer',
      expires_in: 3600,
    });

    const config = new Config(
      'directory.example.com:443',
      Config.DEFAULT_DIRCTL_PATH,
      Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
      'oidc',
      Config.DEFAULT_JWT_AUDIENCE,
      Config.DEFAULT_TLS_CA_FILE,
      Config.DEFAULT_TLS_CERT_FILE,
      Config.DEFAULT_TLS_KEY_FILE,
      '',
      Config.DEFAULT_TLS_SERVER_NAME,
      Config.DEFAULT_TLS_SKIP_VERIFY,
      'https://issuer.example.com',
      'client-id',
    );
    const client = new Client(config);
    await client.authenticateOAuthPkce();

    const cachedToken = new TokenCache().load();
    expect(cachedToken).toBeDefined();
    expect(cachedToken!.accessToken).toBe('fresh-token');
    expect(cachedToken!.refreshToken).toBe('refresh-token');
    expect(cachedToken!.provider).toBe('oidc');
    expect(cachedToken!.issuer).toBe('https://issuer.example.com');
    expect(cachedToken!.tokenType).toBe('bearer');
    expect(cachedToken!.createdAt).toBeDefined();
    expect(cachedToken!.expiresAt).toBeDefined();
  });
});
