import "@nomicfoundation/hardhat-toolbox";

const CHAIN_ID = process.env.CHAIN_ID?.replace(/\D/g, "");

/** @type import('hardhat/config').HardhatUserConfig */
const config = {
  solidity: "0.8.24",
  networks: {
    hardhat: {
      // TODO: https://github.com/NomicFoundation/hardhat/issues/2305
      // We have to reserve a specific chainID for our node, otherwise
      // the contracts when deployed via Hardhat will force default value
      chainId: Number(CHAIN_ID ?? 31337),
    },
  },
};

export default config;
