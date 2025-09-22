export const config = {
    isPrivateChain: process.env.NETWORK_URL ? true : false,
    networkUrl: process.env.NETWORK_URL || "http://127.0.0.1:8545",
    networkChainID: process.env.NETWORK_CHAIN_ID || null,
    networkAccounts: process.env.NETWORK_ACCOUNT_HEX_PK ? [process.env.NETWORK_ACCOUNT_HEX_PK] : "remote"
};
