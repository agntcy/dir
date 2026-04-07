// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { createHash, randomBytes } from 'node:crypto';
import { spawn } from 'node:child_process';
import * as http from 'node:http';
import * as https from 'node:https';
import { platform } from 'node:process';
import { URL } from 'node:url';

export class OAuthPkceError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'OAuthPkceError';
  }
}

/** Subset of client config used by PKCE helpers (avoids circular imports). */
export interface OidcPkceConfig {
  oidcIssuer: string;
  oidcClientId: string;
  oidcClientSecret: string;
  oidcRedirectUri: string;
  oidcCallbackPort: number;
  oidcAuthTimeout: number;
  oidcScopes: string[];
}

export function normalizeIssuer(issuer: string): string {
  const u = issuer.replace(/\/+$/, '');
  if (!u.startsWith('https://') && !u.startsWith('http://')) {
    throw new Error('oidc_issuer must be an absolute URL (https:// recommended)');
  }
  return u;
}

export interface OpenIdConfiguration {
  authorization_endpoint: string;
  token_endpoint: string;
  [key: string]: unknown;
}

async function httpRequest(
  method: string,
  urlStr: string,
  body: string | undefined,
  headers: Record<string, string>,
  options: { timeoutMs: number; rejectUnauthorized: boolean },
): Promise<{ status: number; text: string }> {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const isHttps = u.protocol === 'https:';
    const lib = isHttps ? https : http;
    const port =
      u.port !== ''
        ? Number(u.port)
        : isHttps
          ? 443
          : 80;
    const req = lib.request(
      {
        hostname: u.hostname,
        port,
        path: `${u.pathname}${u.search}`,
        method,
        headers,
        ...(isHttps ? { rejectUnauthorized: options.rejectUnauthorized } : {}),
      },
      (res) => {
        const chunks: Buffer[] = [];
        res.on('data', (c: Buffer) => chunks.push(c));
        res.on('end', () => {
          resolve({
            status: res.statusCode ?? 0,
            text: Buffer.concat(chunks).toString('utf-8'),
          });
        });
      },
    );
    req.setTimeout(options.timeoutMs, () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });
    req.on('error', reject);
    if (body !== undefined) {
      req.write(body);
    }
    req.end();
  });
}

export async function fetchOpenidConfiguration(
  issuer: string,
  options: { verify: boolean; timeoutMs: number },
): Promise<OpenIdConfiguration> {
  const base = normalizeIssuer(issuer);
  const url = `${base}/.well-known/openid-configuration`;
  let response: { status: number; text: string };
  try {
    response = await httpRequest('GET', url, undefined, {}, {
      timeoutMs: options.timeoutMs,
      rejectUnauthorized: options.verify,
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    throw new OAuthPkceError(`OpenID discovery failed: ${msg}`);
  }
  if (response.status < 200 || response.status >= 300) {
    throw new OAuthPkceError(`OpenID discovery HTTP ${response.status}: ${response.text.slice(0, 500)}`);
  }
  let data: Record<string, unknown>;
  try {
    data = JSON.parse(response.text) as Record<string, unknown>;
  } catch {
    throw new OAuthPkceError('OpenID discovery response is not valid JSON');
  }
  if (typeof data.authorization_endpoint !== 'string' || typeof data.token_endpoint !== 'string') {
    throw new OAuthPkceError(
      'OpenID configuration missing authorization_endpoint or token_endpoint',
    );
  }
  return data as OpenIdConfiguration;
}

function urlEncodeForm(body: Record<string, string>): string {
  return new URLSearchParams(body).toString();
}

export async function exchangeAuthorizationCode(
  tokenEndpoint: string,
  params: {
    code: string;
    redirectUri: string;
    clientId: string;
    codeVerifier: string;
    clientSecret: string;
    verify: boolean;
    timeoutMs: number;
  },
): Promise<Record<string, unknown>> {
  const body: Record<string, string> = {
    grant_type: 'authorization_code',
    code: params.code,
    redirect_uri: params.redirectUri,
    client_id: params.clientId,
    code_verifier: params.codeVerifier,
  };
  if (params.clientSecret) {
    body.client_secret = params.clientSecret;
  }
  const encoded = urlEncodeForm(body);
  const response = await httpRequest(
    'POST',
    tokenEndpoint,
    encoded,
    { 'Content-Type': 'application/x-www-form-urlencoded', 'Content-Length': String(Buffer.byteLength(encoded)) },
    { timeoutMs: params.timeoutMs, rejectUnauthorized: params.verify },
  );
  let json: Record<string, unknown>;
  try {
    json = JSON.parse(response.text) as Record<string, unknown>;
  } catch {
    throw new OAuthPkceError(`Token response not JSON: ${response.text.slice(0, 500)}`);
  }
  if (response.status < 200 || response.status >= 300) {
    throw new OAuthPkceError(`Token HTTP ${response.status}: ${response.text.slice(0, 500)}`);
  }
  return json;
}

function base64Url(buf: Buffer): string {
  return buf
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}

function createS256CodeChallenge(verifier: string): string {
  const hash = createHash('sha256').update(verifier, 'utf8').digest();
  return base64Url(hash);
}

function openBrowser(url: string): void {
  const os = platform;
  if (os === 'darwin') {
    spawn('open', [url], { stdio: 'ignore', detached: true }).unref();
  } else if (os === 'win32') {
    spawn('cmd', ['/c', 'start', '""', url], { stdio: 'ignore', detached: true, shell: false }).unref();
  } else {
    spawn('xdg-open', [url], { stdio: 'ignore', detached: true }).unref();
  }
}

export class OAuthTokenHolder {
  private accessToken: string | undefined;

  setTokens(accessToken: string): void {
    this.accessToken = accessToken;
  }

  updateFromTokenResponse(payload: Record<string, unknown>): void {
    const access = payload.access_token;
    if (!access || typeof access !== 'string') {
      throw new OAuthPkceError('Token response missing access_token');
    }
    this.setTokens(access);
  }

  getAccessToken(): string {
    if (this.accessToken === undefined) {
      throw new Error(
        'No OAuth access token: set DIRECTORY_CLIENT_AUTH_TOKEN or call Client.authenticateOAuthPkce()',
      );
    }
    return this.accessToken;
  }
}

export async function runLoopbackPkceLogin(
  config: OidcPkceConfig,
  metadata: OpenIdConfiguration | undefined,
  options: { verify: boolean; timeoutMs: number },
): Promise<Record<string, unknown>> {
  if (!config.oidcIssuer) {
    throw new Error('oidc_issuer is required for OAuth PKCE');
  }
  if (!config.oidcClientId) {
    throw new Error('oidc_client_id is required for OAuth PKCE');
  }

  const redirectUri = config.oidcRedirectUri.trim();
  const parsed = new URL(redirectUri);
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('oidc_redirect_uri must be an absolute http(s) URL');
  }
  if (parsed.hostname !== 'localhost' && parsed.hostname !== '127.0.0.1') {
    throw new Error('loopback PKCE requires redirect host localhost or 127.0.0.1');
  }
  let path = parsed.pathname || '/';
  if (!path.startsWith('/')) {
    path = `/${path}`;
  }

  const port = config.oidcCallbackPort;

  const meta =
    metadata ??
    (await fetchOpenidConfiguration(config.oidcIssuer, {
      verify: options.verify,
      timeoutMs: Math.min(30_000, options.timeoutMs),
    }));

  const authEp = meta.authorization_endpoint;
  const tokenEp = meta.token_endpoint;

  const codeVerifier = base64Url(randomBytes(48));
  const codeChallenge = createS256CodeChallenge(codeVerifier);
  const state = base64Url(randomBytes(24));

  const result: { code?: string } = {};
  const errorHolder: string[] = [];

  let settled = false;
  let resolveDone!: () => void;
  const donePromise = new Promise<void>((r) => {
    resolveDone = r;
  });
  function settle(): void {
    if (!settled) {
      settled = true;
      resolveDone();
    }
  }

  const server = http.createServer((req, res) => {
    try {
      if (!req.url) {
        errorHolder.push('missing url');
        res.writeHead(400);
        res.end();
        return;
      }
      const u = new URL(req.url, `http://127.0.0.1:${port}`);
      if (u.pathname !== path) {
        errorHolder.push('redirect path does not match oidc_redirect_uri');
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
        return;
      }
      const err = u.searchParams.get('error');
      if (err) {
        const desc = u.searchParams.get('error_description') ?? '';
        errorHolder.push(`${err}: ${desc}`);
        const body = Buffer.from(
          '<!DOCTYPE html><html><body><p>Authorization failed. You may close this window.</p></body></html>',
          'utf-8',
        );
        res.writeHead(200, {
          'Content-Type': 'text/html; charset=utf-8',
          'Content-Length': String(body.length),
          'Cache-Control': 'no-store',
        });
        res.end(body);
        return;
      }
      if (u.searchParams.get('state') !== state) {
        errorHolder.push('state mismatch');
        const body = Buffer.from(
          '<!DOCTYPE html><html><body><p>Invalid state. You may close this window.</p></body></html>',
          'utf-8',
        );
        res.writeHead(200, {
          'Content-Type': 'text/html; charset=utf-8',
          'Content-Length': String(body.length),
          'Cache-Control': 'no-store',
        });
        res.end(body);
        return;
      }
      const code = u.searchParams.get('code');
      if (!code) {
        errorHolder.push('missing code');
        const body = Buffer.from(
          '<!DOCTYPE html><html><body><p>Missing code. You may close this window.</p></body></html>',
          'utf-8',
        );
        res.writeHead(200, {
          'Content-Type': 'text/html; charset=utf-8',
          'Content-Length': String(body.length),
          'Cache-Control': 'no-store',
        });
        res.end(body);
        return;
      }
      result.code = code;
      const body = Buffer.from(
        '<!DOCTYPE html><html><body><p>Login successful. You may close this window.</p></body></html>',
        'utf-8',
      );
      res.writeHead(200, {
        'Content-Type': 'text/html; charset=utf-8',
        'Content-Length': String(body.length),
        'Cache-Control': 'no-store',
      });
      res.end(body);
    } finally {
      settle();
    }
  });

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, '127.0.0.1', () => resolve());
  });

  const scopeStr =
    config.oidcScopes && config.oidcScopes.length > 0
      ? config.oidcScopes.join(' ')
      : 'openid';
  const authParams = new URLSearchParams({
    response_type: 'code',
    client_id: config.oidcClientId,
    redirect_uri: redirectUri,
    scope: scopeStr,
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  });
  const sep = authEp.includes('?') ? '&' : '?';
  const authorizeUrl = `${authEp}${sep}${authParams.toString()}`;

  openBrowser(authorizeUrl);

  const waitMs = config.oidcAuthTimeout * 1000;
  try {
    await Promise.race([
      donePromise,
      new Promise<never>((_, rej) => {
        setTimeout(() => {
          rej(new OAuthPkceError(`OAuth callback timed out after ${config.oidcAuthTimeout}s`));
        }, waitMs);
      }),
    ]);
  } finally {
    await new Promise<void>((r) => {
      server.close(() => r());
    });
  }

  if (errorHolder.length > 0) {
    throw new OAuthPkceError(errorHolder[0] ?? 'unknown error');
  }
  const code = result.code;
  if (!code) {
    throw new OAuthPkceError('Authorization did not return a code');
  }

  return exchangeAuthorizationCode(tokenEp, {
    code,
    redirectUri: redirectUri,
    clientId: config.oidcClientId,
    codeVerifier,
    clientSecret: config.oidcClientSecret,
    verify: options.verify,
    timeoutMs: Math.min(30_000, options.timeoutMs),
  });
}
