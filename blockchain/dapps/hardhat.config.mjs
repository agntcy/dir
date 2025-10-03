import "@nomicfoundation/hardhat-toolbox";
import { config } from "./config/config";
import fs from "fs";
import { task } from "hardhat/config";

// test task overrides test to run pre- and post- hooks
task("test", async (taskArgs, hre, runSuper) => {
  console.log("Before running the tests...");
  const result = await runSuper();
  console.log("After running the tests...");
  return result;
});

// deploy task deploys contracts and prints the deployment details as a JSON object.
// Example: npm run deploy -- --output-file "123.json"
task("deploy", "Deploy all contracts")
  .addOptionalParam("outputFile", "Filepath to save deployment info. Prints to STDOUT if not specified")
  .setAction(async (taskArgs, hre, runSuper) => {
    // Deploy Store contract
    const AgentStoreModule = (await import("./ignition/modules/AgentStore.js")).default;
    const { store } = await hre.ignition.deploy(AgentStoreModule);
    var storeAddress = await store.getAddress();

    // Create deployment info
    const deploymentInfo = JSON.stringify({
      networkName: hre.network.name,
      networkAddress: hre.network.config.url,
      chainId: Number(await hre.network.provider.send('eth_chainId')),
      storeContract: storeAddress
    }, null, 2);

    // Print deployment info
    console.log(deploymentInfo);

    // Save to a file
    if (taskArgs.outputFile) {
      fs.writeFileSync(taskArgs.outputFile, deploymentInfo);
    }
  });

/** @type import('hardhat/config').HardhatUserConfig */
const userConfig = {
  solidity: "0.8.24",
  defaultNetwork: config.isPrivateChain ? "appchain" : "hardhat",
  networks: {
    appchain: {
      url: config.networkUrl,
      // chainID value is ignored when we are running against Hardhat node
      // and will be read from the node directly.
      chainID: config.networkChainID,
      accounts: config.networkAccounts
    }
  },
  // gobind config removed
};

export default userConfig;
