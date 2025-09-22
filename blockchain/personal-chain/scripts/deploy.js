
import pkg from "hardhat";
const { ethers } = pkg;

async function main() {
  const Store = await ethers.getContractFactory("AgentStore");
  const store = await Store.deploy();
  await store.waitForDeployment();
  console.log(`AgentStore deployed to: ${store.target}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
