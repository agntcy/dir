const {
  time,
  loadFixture,
} = require("@nomicfoundation/hardhat-toolbox/network-helpers");
const { anyValue } = require("@nomicfoundation/hardhat-chai-matchers/withArgs");
const { expect } = require("chai");

describe("AgentStore", function () {
  // We define a fixture to reuse the same setup in every test.
  // We use loadFixture to run this setup once, snapshot that state,
  // and reset Hardhat Network to that snapshot in every test.
  async function deploy() {
    // Contracts are deployed using the first signer/account by default
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
