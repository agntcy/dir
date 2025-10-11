import { loadFixture } from "@nomicfoundation/hardhat-network-helpers";
import { expect } from "chai";

describe("AgentStore", function () {
  async function deploy() {
    const [owner, otherAccount] = await ethers.getSigners();
    const Store = await ethers.getContractFactory("AgentStore");
    const store = await Store.deploy();
    return { store, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should deploy with default state", async function () {
      const { store } = await loadFixture(deploy);
      expect(await store.total()).to.equal(0);
    });
  });
});
