import { buildModule } from "@nomicfoundation/hardhat-ignition/modules";

export default buildModule("AgentStoreModule", (m) => {
  const store = m.contract("AgentStore");
  return { store };
});
