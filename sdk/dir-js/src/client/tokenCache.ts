// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { chmodSync, existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { env } from 'node:process';

export const DEFAULT_TOKEN_CACHE_DIR = 'dirctl';
export const TOKEN_CACHE_FILE = 'auth-token.json';

const DEFAULT_TOKEN_VALIDITY_MS = 8 * 60 * 60 * 1000;
const TOKEN_EXPIRY_BUFFER_MS = 5 * 60 * 1000;
const CACHE_DIR_PERMS = 0o700;
const CACHE_FILE_PERMS = 0o600;

function utcNow(): Date {
  return new Date();
}

function parseTimestamp(value: string | undefined | null): Date | undefined {
  if (!value) {
    return undefined;
  }
  const normalized = value.replace('Z', '+00:00');
  const d = new Date(normalized);
  return Number.isNaN(d.getTime()) ? undefined : d;
}

function formatTimestamp(value: Date | undefined): string | undefined {
  if (value === undefined) {
    return undefined;
  }
  return value.toISOString();
}

export interface CachedTokenJson {
  access_token: string;
  token_type?: string;
  provider?: string;
  issuer?: string;
  refresh_token?: string;
  expires_at?: string;
  user?: string;
  user_id?: string;
  email?: string;
  created_at?: string;
}

export class CachedToken {
  accessToken: string;
  tokenType: string;
  provider: string;
  issuer: string;
  refreshToken: string;
  expiresAt: Date | undefined;
  user: string;
  userId: string;
  email: string;
  createdAt: Date | undefined;

  constructor(
    accessToken: string,
    tokenType = '',
    provider = '',
    issuer = '',
    refreshToken = '',
    expiresAt: Date | undefined = undefined,
    user = '',
    userId = '',
    email = '',
    createdAt: Date | undefined = undefined,
  ) {
    this.accessToken = accessToken;
    this.tokenType = tokenType;
    this.provider = provider;
    this.issuer = issuer;
    this.refreshToken = refreshToken;
    this.expiresAt = expiresAt;
    this.user = user;
    this.userId = userId;
    this.email = email;
    this.createdAt = createdAt;
  }

  static fromJson(payload: CachedTokenJson): CachedToken {
    return new CachedToken(
      String(payload.access_token ?? ''),
      String(payload.token_type ?? ''),
      String(payload.provider ?? ''),
      String(payload.issuer ?? ''),
      String(payload.refresh_token ?? ''),
      parseTimestamp(payload.expires_at),
      String(payload.user ?? ''),
      String(payload.user_id ?? ''),
      String(payload.email ?? ''),
      parseTimestamp(payload.created_at),
    );
  }

  toJson(): Record<string, string> {
    const payload: Record<string, string> = {
      access_token: this.accessToken,
      created_at: formatTimestamp(this.createdAt ?? utcNow()) ?? '',
    };
    if (this.tokenType) {
      payload.token_type = this.tokenType;
    }
    if (this.provider) {
      payload.provider = this.provider;
    }
    if (this.issuer) {
      payload.issuer = this.issuer;
    }
    if (this.refreshToken) {
      payload.refresh_token = this.refreshToken;
    }
    const exp = formatTimestamp(this.expiresAt);
    if (exp !== undefined) {
      payload.expires_at = exp;
    }
    if (this.user) {
      payload.user = this.user;
    }
    if (this.userId) {
      payload.user_id = this.userId;
    }
    if (this.email) {
      payload.email = this.email;
    }
    return payload;
  }
}

export class TokenCache {
  readonly cacheDir: string;

  constructor(cacheDir?: string) {
    if (cacheDir !== undefined) {
      this.cacheDir = cacheDir;
    } else {
      const configHome = env['XDG_CONFIG_HOME'];
      const baseDir = configHome ? configHome : join(homedir(), '.config');
      this.cacheDir = join(baseDir, DEFAULT_TOKEN_CACHE_DIR);
    }
  }

  getCachePath(): string {
    return join(this.cacheDir, TOKEN_CACHE_FILE);
  }

  load(): CachedToken | undefined {
    const path = this.getCachePath();
    if (!existsSync(path)) {
      return undefined;
    }
    const raw = readFileSync(path, 'utf-8');
    const payload = JSON.parse(raw) as CachedTokenJson;
    return CachedToken.fromJson(payload);
  }

  save(token: CachedToken): void {
    mkdirSync(this.cacheDir, { mode: CACHE_DIR_PERMS, recursive: true });
    try {
      chmodSync(this.cacheDir, CACHE_DIR_PERMS);
    } catch {
      // ignore chmod failures (e.g. Windows)
    }
    if (token.createdAt === undefined) {
      token.createdAt = utcNow();
    }
    const serialized = `${JSON.stringify(token.toJson(), null, 2)}\n`;
    const path = this.getCachePath();
    writeFileSync(path, serialized, { encoding: 'utf-8', mode: CACHE_FILE_PERMS });
    try {
      chmodSync(path, CACHE_FILE_PERMS);
    } catch {
      // ignore chmod failures
    }
  }

  clear(): void {
    const path = this.getCachePath();
    if (existsSync(path)) {
      unlinkSync(path);
    }
  }

  isValid(token: CachedToken | undefined): boolean {
    if (token === undefined || !token.accessToken) {
      return false;
    }
    const now = utcNow().getTime();
    if (token.expiresAt === undefined) {
      const createdAt = token.createdAt ?? new Date(now);
      return now < createdAt.getTime() + DEFAULT_TOKEN_VALIDITY_MS;
    }
    return now + TOKEN_EXPIRY_BUFFER_MS < token.expiresAt.getTime();
  }

  getValidToken(): CachedToken | undefined {
    const token = this.load();
    if (!this.isValid(token)) {
      return undefined;
    }
    return token;
  }
}
