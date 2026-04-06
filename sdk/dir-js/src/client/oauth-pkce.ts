// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { createHash, randomBytes } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import * as http from 'node:http';
import * as https from 'node:https';
import { spawn } from 'node:child_process';
import { URL } from 'node:url';

/** Fields required for OAuth PKCE / client-credentials against an OIDC issuer (e.g. DeX). */
export interface OauthPkceConfig {
  oidcIssuer: string;
  oidcClientId: string;
  oidcClientSecret: string;
  oidcRedirectUri: string;
  oidcCallbackPort: number;
  oidcAuthTimeout: number;
  oidcScopes: string[];
  tlsSkipVerify: boolean;
  oidcAccessToken: string;
  oidcMachineClientId: string;
  oidcMachineClientSecret: string;
  oidcMachineClientSecretFile: string;
  oidcMachineScopes: string[];
  oidcMachineTokenEndpoint: string;
}

export class OAuthPkceError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'OAuthPkceError';
  }
}

export function normalizeIssuer(issuer: string): string {
  const u = issuer.trim().replace(/\/+$/, '');
  if (!u.startsWith('https://') && !u.startsWith('http://')) {
    throw new Error('oidcIssuer must be an absolute URL (https:// recommended)');
  }
  return u;
}

function httpTimeoutMs(config: OauthPkceConfig): number {
  return Math.min(30_000, Math.max(1, config.oidcAuthTimeout) * 1000);
}

function parseJsonResponse(
  statusCode: number,
  body: string,
): Record<string, unknown> {
  let payload: Record<string, unknown> = {};
  try {
    payload = JSON.parse(body) as Record<string, unknown>;
  } catch {
    payload = {};
  }
  if (statusCode < 200 || statusCode >= 300) {
    const err = payload['error'];
    const desc = payload['error_description'];
    if (typeof err === 'string') {
      throw new OAuthPkceError(
        `Token HTTP ${statusCode}: ${err} (${typeof desc === 'string' ? desc : 'no description'})`,
      );
    }
    throw new OAuthPkceError(
      `Token HTTP ${statusCode}: ${body.slice(0, 500)}`,
    );
  }
  return payload;
}

function requestUrl(
  urlStr: string,
  options: {
    method: string;
    headers: Record<string, string>;
    body?: string;
    rejectUnauthorized: boolean;
    timeoutMs: number;
  },
): Promise<{ statusCode: number; body: string }> {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const lib = u.protocol === 'https:' ? https : http;
    const reqOptions: http.RequestOptions = {
      protocol: u.protocol,
      hostname: u.hostname,
      port: u.port || (u.protocol === 'https:' ? 443 : 80),
      path: `${u.pathname}${u.search}`,
      method: options.method,
      headers: options.headers,
    };
    if (u.protocol === 'https:') {
      (reqOptions as https.RequestOptions).rejectUnauthorized =
        options.rejectUnauthorized;
    }

    const req = lib.request(reqOptions, (res) => {
      const chunks: Buffer[] = [];
      res.on('data', (c) => chunks.push(c as Buffer));
      res.on('end', () => {
        clearTimeout(timer);
        resolve({
          statusCode: res.statusCode ?? 0,
          body: Buffer.concat(chunks).toString('utf8'),
        });
      });
    });

    const timer = setTimeout(() => {
      req.destroy();
      reject(new OAuthPkceError(`Request timed out: ${urlStr}`));
    }, options.timeoutMs);

    req.on('error', (e) => {
      clearTimeout(timer);
      reject(e);
    });
    if (options.body !== undefined) {
      req.write(options.body);
    }
    req.end();
  });
}

export async function fetchOpenIdConfiguration(
  issuer: string,
  options: { verifyTls: boolean; timeoutMs: number },
): Promise<Record<string, unknown>> {
  const base = normalizeIssuer(issuer);
  const url = `${base}/.well-known/openid-configuration`;
  const { statusCode, body } = await requestUrl(url, {
    method: 'GET',
    headers: { Accept: 'application/json' },
    rejectUnauthorized: options.verifyTls,
    timeoutMs: options.timeoutMs,
  });
  if (statusCode < 200 || statusCode >= 300) {
    throw new OAuthPkceError(
      `OpenID discovery failed: HTTP ${statusCode} ${body.slice(0, 200)}`,
    );
  }
  const data = JSON.parse(body) as Record<string, unknown>;
  if (typeof data['authorization_endpoint'] !== 'string' || typeof data['token_endpoint'] !== 'string') {
    throw new OAuthPkceError(
      'OpenID configuration missing authorization_endpoint or token_endpoint',
    );
  }
  return data;
}

async function formPost(
  urlStr: string,
  body: Record<string, string>,
  verifyTls: boolean,
  timeoutMs: number,
): Promise<Record<string, unknown>> {
  const encoded = new URLSearchParams(body).toString();
  const { statusCode, body: resBody } = await requestUrl(urlStr, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Content-Length': String(Buffer.byteLength(encoded)),
    },
    body: encoded,
    rejectUnauthorized: verifyTls,
    timeoutMs,
  });
  return parseJsonResponse(statusCode, resBody);
}

export function createPkcePair(): { verifier: string; challenge: string } {
  const verifier = randomBytes(48).toString('base64url');
  const challenge = createHash('sha256').update(verifier).digest('base64url');
  return { verifier, challenge };
}

export async function exchangeAuthorizationCode(
  tokenEndpoint: string,
  params: {
    code: string;
    redirectUri: string;
    clientId: string;
    codeVerifier: string;
    clientSecret: string;
    verifyTls: boolean;
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
  return formPost(tokenEndpoint, body, params.verifyTls, params.timeoutMs);
}

function openBrowser(url: string): void {
  const platform = process.platform;
  let cmd: string;
  let args: string[];
  if (platform === 'darwin') {
    cmd = 'open';
    args = [url];
  } else if (platform === 'win32') {
    cmd = 'cmd';
    args = ['/c', 'start', '""', url];
  } else {
    cmd = 'xdg-open';
    args = [url];
  }
  const child = spawn(cmd, args, { detached: true, stdio: 'ignore' });
  child.unref();
}

/**
 * Run browser authorization with PKCE; local HTTP server receives redirect on 127.0.0.1.
 */
export async function runLoopbackPkceLogin(
  config: OauthPkceConfig,
  metadata?: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  if (!config.oidcIssuer.trim()) {
    throw new Error('oidcIssuer is required for OAuth PKCE');
  }
  if (!config.oidcClientId.trim()) {
    throw new Error('oidcClientId is required for OAuth PKCE');
  }

  const redirectUri = config.oidcRedirectUri.trim();
  let parsed: URL;
  try {
    parsed = new URL(redirectUri);
  } catch {
    throw new OAuthPkceError('oidcRedirectUri must be an absolute http(s) URL');
  }
  if (
    (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') ||
    !parsed.hostname
  ) {
    throw new OAuthPkceError('oidcRedirectUri must be an absolute http(s) URL');
  }
  if (parsed.hostname !== 'localhost' && parsed.hostname !== '127.0.0.1') {
    throw new OAuthPkceError(
      'loopback PKCE requires redirect host localhost or 127.0.0.1',
    );
  }
  let path = parsed.pathname || '/';
  if (!path.startsWith('/')) {
    path = `/${path}`;
  }

  const port = config.oidcCallbackPort;
  const verifyTls = !config.tlsSkipVerify;
  const timeoutMs = httpTimeoutMs(config);

  let meta = metadata;
  if (!meta) {
    meta = await fetchOpenIdConfiguration(config.oidcIssuer, {
      verifyTls,
      timeoutMs,
    });
  }
  const authEp = String(meta['authorization_endpoint']);
  const tokenEp = String(meta['token_endpoint']);

  const { verifier: codeVerifier, challenge: codeChallenge } = createPkcePair();
  const state = randomBytes(24).toString('base64url');

  const scopeStr =
    config.oidcScopes.length > 0
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

  const result: { code?: string } = {};
  const errors: string[] = [];
  let doneResolve!: () => void;
  const done = new Promise<void>((r) => {
    doneResolve = r;
  });

  const server = createServer((req: IncomingMessage, res: ServerResponse) => {
    try {
      const reqUrl = new URL(req.url || '/', 'http://127.0.0.1');
      if (reqUrl.pathname !== path) {
        errors.push('redirect path does not match oidcRedirectUri');
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not Found');
        return;
      }
      const qs = new URLSearchParams(reqUrl.search);
      if (qs.get('error')) {
        const err = qs.get('error') || 'unknown';
        const desc = qs.get('error_description') || '';
        errors.push(`${err}: ${desc}`);
        sendOkPage(res, 'Authorization failed. You may close this window.');
        return;
      }
      if (qs.get('state') !== state) {
        errors.push('state mismatch');
        sendOkPage(res, 'Invalid state. You may close this window.');
        return;
      }
      const code = qs.get('code');
      if (!code) {
        errors.push('missing code');
        sendOkPage(res, 'Missing code. You may close this window.');
        return;
      }
      result.code = code;
      sendOkPage(res, 'Login successful. You may close this window.');
    } finally {
      doneResolve();
    }
  });

  function sendOkPage(res: ServerResponse, message: string): void {
    const body = Buffer.from(
      `<!DOCTYPE html><html><body><p>${escapeHtml(message)}</p></body></html>`,
      'utf8',
    );
    res.writeHead(200, {
      'Content-Type': 'text/html; charset=utf-8',
      'Content-Length': String(body.length),
      'Cache-Control': 'no-store',
    });
    res.end(body);
  }

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, '127.0.0.1', () => resolve());
  });

  openBrowser(authorizeUrl);

  const waitMs = Math.max(1, config.oidcAuthTimeout) * 1000;
  const timedOut = await Promise.race([
    done.then(() => false),
    new Promise<boolean>((r) => setTimeout(() => r(true), waitMs)),
  ]);

  await new Promise<void>((resolve, reject) => {
    server.close((err) => (err ? reject(err) : resolve()));
  });

  if (timedOut) {
    throw new OAuthPkceError(
      `OAuth callback timed out after ${config.oidcAuthTimeout}s`,
    );
  }
  if (errors.length > 0) {
    throw new OAuthPkceError(errors[0]!);
  }
  if (!result.code) {
    throw new OAuthPkceError('Authorization did not return a code');
  }

  return exchangeAuthorizationCode(tokenEp, {
    code: result.code,
    redirectUri,
    clientId: config.oidcClientId,
    codeVerifier,
    clientSecret: config.oidcClientSecret,
    verifyTls,
    timeoutMs,
  });
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export class OAuthTokenHolder {
  private _tokenEndpoint = '';
  private readonly _clientId: string;
  private readonly _clientSecret: string;
  private readonly _verifyTls: boolean;
  private readonly _httpTimeoutMs: number;
  private _accessToken: string | null = null;
  private _refreshToken: string | null = null;
  private _expiresAtMonotonic: number | null = null;
  private _refreshPromise: Promise<void> | null = null;

  constructor(
    tokenEndpoint: string,
    clientId: string,
    clientSecret: string,
    options: { verifyTls: boolean; httpTimeoutMs: number },
  ) {
    this._tokenEndpoint = tokenEndpoint;
    this._clientId = clientId;
    this._clientSecret = clientSecret;
    this._verifyTls = options.verifyTls;
    this._httpTimeoutMs = options.httpTimeoutMs;
  }

  setTokenEndpoint(endpoint: string): void {
    this._tokenEndpoint = endpoint;
  }

  setTokens(
    accessToken: string,
    options?: { refreshToken?: string | null; expiresIn?: number | null },
  ): void {
    if (!accessToken) {
      this._accessToken = null;
      this._refreshToken = null;
      this._expiresAtMonotonic = null;
      return;
    }
    this._accessToken = accessToken;
    this._refreshToken = options?.refreshToken ?? null;
    const exp = options?.expiresIn;
    if (typeof exp === 'number' && Number.isFinite(exp)) {
      this._expiresAtMonotonic = performance.now() + exp * 1000;
    } else {
      this._expiresAtMonotonic = null;
    }
  }

  updateFromTokenResponse(payload: Record<string, unknown>): void {
    const access = payload['access_token'];
    if (typeof access !== 'string' || !access) {
      throw new OAuthPkceError('Token response missing access_token');
    }
    const refresh = payload['refresh_token'];
    const refreshS = typeof refresh === 'string' ? refresh : null;
    const expiresIn = payload['expires_in'];
    const expI =
      typeof expiresIn === 'number'
        ? expiresIn
        : typeof expiresIn === 'string'
          ? parseInt(expiresIn, 10)
          : null;
    this.setTokens(access, {
      refreshToken: refreshS,
      expiresIn: Number.isFinite(expI as number) ? (expI as number) : null,
    });
  }

  async getAccessToken(): Promise<string> {
    if (this._accessToken === null || this._accessToken === '') {
      throw new Error(
        'No OAuth access token: set DIRECTORY_CLIENT_OIDC_ACCESS_TOKEN or complete OAuth login',
      );
    }
    if (this._shouldRefresh()) {
      if (this._refreshPromise) {
        await this._refreshPromise;
      } else {
        this._refreshPromise = this.doRefresh();
        try {
          await this._refreshPromise;
        } finally {
          this._refreshPromise = null;
        }
      }
    }
    return this._accessToken!;
  }

  private _shouldRefresh(): boolean {
    if (!this._refreshToken || this._expiresAtMonotonic === null) {
      return false;
    }
    return performance.now() >= this._expiresAtMonotonic - 60_000;
  }

  private async doRefresh(): Promise<void> {
    if (!this._refreshToken || !this._tokenEndpoint) {
      return;
    }
    const body: Record<string, string> = {
      grant_type: 'refresh_token',
      refresh_token: this._refreshToken,
      client_id: this._clientId,
    };
    if (this._clientSecret) {
      body.client_secret = this._clientSecret;
    }
    const payload = await formPost(
      this._tokenEndpoint,
      body,
      this._verifyTls,
      this._httpTimeoutMs,
    );
    this.updateFromTokenResponse(payload);
  }
}

function hasMachineOidcConfig(config: OauthPkceConfig): boolean {
  const secret =
    config.oidcMachineClientSecret.trim() ||
    config.oidcMachineClientSecretFile.trim();
  return (
    Boolean(config.oidcIssuer.trim()) &&
    Boolean(config.oidcMachineClientId.trim()) &&
    Boolean(secret)
  );
}

function resolveMachineClientSecret(config: OauthPkceConfig): string {
  let secret = config.oidcMachineClientSecret.trim();
  if (!secret && config.oidcMachineClientSecretFile.trim()) {
    try {
      secret = readFileSync(config.oidcMachineClientSecretFile.trim(), 'utf8').trim();
    } catch (e) {
      throw new Error(
        `Failed to read oidcMachineClientSecretFile: ${(e as Error).message}`,
      );
    }
  }
  if (!secret) {
    throw new Error(
      'oidcMachineClientSecret is required for client credentials flow',
    );
  }
  return secret;
}

async function resolveMachineTokenEndpoint(
  config: OauthPkceConfig,
  verifyTls: boolean,
  timeoutMs: number,
): Promise<string> {
  const explicit = config.oidcMachineTokenEndpoint.trim();
  if (explicit) {
    return explicit;
  }
  const meta = await fetchOpenIdConfiguration(config.oidcIssuer, {
    verifyTls,
    timeoutMs,
  });
  return String(meta['token_endpoint']);
}

/**
 * OAuth2 client_credentials grant (e.g. ZITADEL machine user).
 */
export async function runClientCredentialsFlow(
  config: OauthPkceConfig,
  holder: OAuthTokenHolder,
): Promise<void> {
  if (!config.oidcIssuer.trim()) {
    throw new Error('oidcIssuer is required for client credentials flow');
  }
  if (!config.oidcMachineClientId.trim()) {
    throw new Error('oidcMachineClientId is required for client credentials flow');
  }
  const clientSecret = resolveMachineClientSecret(config);
  const verifyTls = !config.tlsSkipVerify;
  const timeoutMs = httpTimeoutMs(config);
  const tokenEndpoint = await resolveMachineTokenEndpoint(
    config,
    verifyTls,
    timeoutMs,
  );

  const body: Record<string, string> = {
    grant_type: 'client_credentials',
    client_id: config.oidcMachineClientId,
    client_secret: clientSecret,
  };
  if (config.oidcMachineScopes.length > 0) {
    body.scope = config.oidcMachineScopes.join(' ');
  }

  const payload = await formPost(
    tokenEndpoint,
    body,
    verifyTls,
    timeoutMs,
  );
  holder.setTokenEndpoint(tokenEndpoint);
  holder.updateFromTokenResponse(payload);
}

/**
 * Create holder, optional discovery for token endpoint, then seed tokens per env (access token, machine, or PKCE).
 */
export async function bootstrapOAuthTokenHolder(
  config: OauthPkceConfig,
): Promise<OAuthTokenHolder> {
  const verifyTls = !config.tlsSkipVerify;
  const timeoutMs = httpTimeoutMs(config);

  let tokenEp = '';
  if (config.oidcIssuer.trim()) {
    try {
      const meta = await fetchOpenIdConfiguration(config.oidcIssuer, {
        verifyTls,
        timeoutMs,
      });
      tokenEp = String(meta['token_endpoint']);
    } catch {
      // Discovery failed; refresh disabled until authenticateOauthPkce sets endpoint
      tokenEp = '';
    }
  }

  const holder = new OAuthTokenHolder(tokenEp, config.oidcClientId, config.oidcClientSecret, {
    verifyTls,
    httpTimeoutMs: timeoutMs,
  });

  const accessFromEnv = config.oidcAccessToken.trim();
  if (accessFromEnv) {
    holder.setTokens(accessFromEnv);
    return holder;
  }

  if (hasMachineOidcConfig(config)) {
    await runClientCredentialsFlow(config, holder);
    return holder;
  }

  if (!config.oidcIssuer.trim()) {
    throw new Error('oidcIssuer is required for OAuth PKCE');
  }
  if (!config.oidcClientId.trim()) {
    throw new Error('oidcClientId is required for OAuth PKCE');
  }

  const meta = await fetchOpenIdConfiguration(config.oidcIssuer, {
    verifyTls,
    timeoutMs,
  });
  holder.setTokenEndpoint(String(meta['token_endpoint']));
  const payload = await runLoopbackPkceLogin(config, meta);
  holder.updateFromTokenResponse(payload);
  return holder;
}

export async function authenticateOauthPkceFlow(
  config: OauthPkceConfig,
  holder: OAuthTokenHolder,
): Promise<void> {
  if (!config.oidcIssuer.trim()) {
    throw new Error('oidcIssuer is required for authenticateOauthPkce');
  }
  if (!config.oidcClientId.trim()) {
    throw new Error('oidcClientId is required for authenticateOauthPkce');
  }
  const verifyTls = !config.tlsSkipVerify;
  const timeoutMs = httpTimeoutMs(config);
  const meta = await fetchOpenIdConfiguration(config.oidcIssuer, {
    verifyTls,
    timeoutMs,
  });
  holder.setTokenEndpoint(String(meta['token_endpoint']));
  const payload = await runLoopbackPkceLogin(config, meta);
  holder.updateFromTokenResponse(payload);
}
