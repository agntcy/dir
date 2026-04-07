// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

/**
 * Interactive OIDC example: runs SearchCIDs only (parity with example_interactive_oidc.py).
 *
 * Requires DIRECTORY_CLIENT_OIDC_CLIENT_ID. Optional: DIRECTORY_CLIENT_OIDC_CLIENT_SECRET,
 * DIRECTORY_CLIENT_SERVER_ADDRESS, DIRECTORY_CLIENT_TLS_SERVER_NAME, DIRECTORY_CLIENT_OIDC_REDIRECT_URI,
 * DIRECTORY_CLIENT_OIDC_CALLBACK_PORT, DIRECTORY_CLIENT_OIDC_AUTH_TIMEOUT, DIRECTORY_CLIENT_AUTH_TOKEN,
 * DIRECTORY_CLIENT_TLS_SKIP_VERIFY.
 */

import { Client, Config, OAuthPkceError, TokenCache, models } from "agntcy-dir";

const DEFAULT_OIDC_ISSUER = "https://dev.idp.ads.outshift.io";
const DEFAULT_SERVER_ADDRESS = "dev.gateway.ads.outshift.io:443";
const DEFAULT_TLS_SERVER_NAME = "dev.gateway.ads.outshift.io";
const DEFAULT_REDIRECT_URI = "http://localhost:8484/callback";

function requireEnv(name) {
  const value = (process.env[name] ?? "").trim();
  if (!value) {
    throw new Error(`${name} is required for the interactive OIDC example`);
  }
  return value;
}

/** Same truthy strings as Config.loadFromEnv (DIRECTORY_CLIENT_TLS_SKIP_VERIFY). */
function parseBoolEnv(value, defaultVal = false) {
  if (value === undefined || value === "") {
    return defaultVal;
  }
  return ["1", "true", "yes", "on"].includes(value.trim().toLowerCase());
}

function parseArgs(argv) {
  const out = { version: "v1*", limit: 3 };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--version" && argv[i + 1]) {
      out.version = argv[++i];
    } else if (a === "--limit" && argv[i + 1]) {
      const n = Number.parseInt(argv[++i], 10);
      if (Number.isFinite(n)) {
        out.limit = n;
      }
    }
  }
  return out;
}

function hasUsableOidcTokenWithoutPkce() {
  const authToken = (process.env.DIRECTORY_CLIENT_AUTH_TOKEN ?? "").trim();
  if (authToken) {
    return true;
  }
  return new TokenCache().getValidToken() !== undefined;
}

function parseOidcCallbackPort() {
  const raw = process.env.DIRECTORY_CLIENT_OIDC_CALLBACK_PORT;
  if (raw === undefined || raw === "") {
    return Config.DEFAULT_OIDC_CALLBACK_PORT;
  }
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) ? n : Config.DEFAULT_OIDC_CALLBACK_PORT;
}

function parseOidcAuthTimeout() {
  const raw = process.env.DIRECTORY_CLIENT_OIDC_AUTH_TIMEOUT;
  if (raw === undefined || raw === "") {
    return Config.DEFAULT_OIDC_AUTH_TIMEOUT;
  }
  const n = Number.parseFloat(raw);
  return Number.isFinite(n) ? n : Config.DEFAULT_OIDC_AUTH_TIMEOUT;
}

function buildConfig() {
  return new Config(
    process.env.DIRECTORY_CLIENT_SERVER_ADDRESS ?? DEFAULT_SERVER_ADDRESS,
    Config.DEFAULT_DIRCTL_PATH,
    Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
    "oidc",
    Config.DEFAULT_JWT_AUDIENCE,
    Config.DEFAULT_TLS_CA_FILE,
    Config.DEFAULT_TLS_CERT_FILE,
    Config.DEFAULT_TLS_KEY_FILE,
    (process.env.DIRECTORY_CLIENT_AUTH_TOKEN ?? "").trim(),
    process.env.DIRECTORY_CLIENT_TLS_SERVER_NAME ?? DEFAULT_TLS_SERVER_NAME,
    parseBoolEnv(process.env.DIRECTORY_CLIENT_TLS_SKIP_VERIFY, false),
    DEFAULT_OIDC_ISSUER,
    requireEnv("DIRECTORY_CLIENT_OIDC_CLIENT_ID"),
    process.env.DIRECTORY_CLIENT_OIDC_CLIENT_SECRET ?? "",
    process.env.DIRECTORY_CLIENT_OIDC_REDIRECT_URI ?? DEFAULT_REDIRECT_URI,
    parseOidcCallbackPort(),
    parseOidcAuthTimeout(),
    undefined,
    undefined,
  );
}

async function buildClient() {
  const config = buildConfig();
  const client = new Client(config);

  if (hasUsableOidcTokenWithoutPkce()) {
    console.log("Using cached OIDC token.");
    return client;
  }

  console.log("No cached OIDC token found. Starting interactive login.");
  await client.authenticateOAuthPkce();
  return client;
}

(async () => {
  try {
    const args = parseArgs(process.argv);
    const client = await buildClient();

    const objects = await client.searchCIDs({
      queries: [
        {
          type: models.search_v1.RecordQueryType.VERSION,
          value: args.version,
        },
      ],
      limit: args.limit,
    });

    console.log(`SearchCIDs results for version ${JSON.stringify(args.version)}:`);
    for (const obj of objects) {
      console.log(obj);
    }
  } catch (e) {
    if (e instanceof OAuthPkceError) {
      console.error(`Interactive OIDC login failed: ${e.message}`);
    }
    throw e;
  }
})();
