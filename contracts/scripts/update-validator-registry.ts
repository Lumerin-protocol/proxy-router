import { requireEnvsSet } from "../lib/utils";
import { viem } from "hardhat";
import { verifyContract } from "./lib/verify";
import { upgrades } from "hardhat";

// https://forum.openzeppelin.com/t/openzeppelin-upgrades-step-by-step-tutorial-for-hardhat/3580
async function main() {
  console.log("ValidatorRegistry update script");
  console.log();

  const env = <
    {
      VALIDATOR_REGISTRY_ADDRESS: `0x${string}`;
    }
  >requireEnvsSet("VALIDATOR_REGISTRY_ADDRESS");

  const pc = await viem.getPublicClient();
  const [deployer] = await viem.getWalletClients();
  console.log("Deployer:", deployer.account.address);

  const ValidatorRegistryProxy = await viem.getContractAt(
    "ValidatorRegistry",
    env.VALIDATOR_REGISTRY_ADDRESS
  );
  const currentImplementation = await upgrades.erc1967.getImplementationAddress(
    env.VALIDATOR_REGISTRY_ADDRESS
  );
  console.log("ValidatorRegistry proxy:", env.VALIDATOR_REGISTRY_ADDRESS);
  console.log("Current implementation:", currentImplementation);

  // Deploy new implementation manually
  console.log("\nDeploying new ValidatorRegistry implementation...");
  const newImpl = await viem.deployContract("ValidatorRegistry");
  console.log("New implementation deployed at:", newImpl.address);
  await verifyContract(newImpl.address, []);

  // Get the proxy admin address and perform manual upgrade
  console.log("\nPerforming manual upgrade...");

  // Upgrade the proxy to point to new implementation
  console.log("Upgrading proxy to new implementation...");
  const txhash = await ValidatorRegistryProxy.write.upgradeToAndCall([newImpl.address, "0x"], {
    account: deployer.account.address,
  });
  await pc.waitForTransactionReceipt({ hash: txhash });

  // Verify the update was successful
  const newImplementation = await upgrades.erc1967.getImplementationAddress(
    env.VALIDATOR_REGISTRY_ADDRESS
  );
  console.log("\nVerification:");
  console.log("New implementation address:", newImplementation);
  console.log(
    "Update successful:",
    newImpl.address.toLowerCase() === newImplementation.toLowerCase()
  );

  console.log("\n---");
  console.log("SUCCESS");
  console.log("New ValidatorRegistry implementation address:", newImpl.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
