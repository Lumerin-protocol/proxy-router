import { viem } from "hardhat";
import { requireEnvsSet } from "../lib/utils";
import { encodeFunctionData } from "viem";
import { writeAndWait } from "./lib/writeContract";
import { verifyContract } from "./lib/verify";

async function main() {
  console.log("Hashrate Oracle deployment script");
  console.log();

  const env = <
    {
      OWNER_PRIVATEKEY: `0x${string}`;
      LUMERIN_TOKEN_ADDRESS: `0x${string}`;
      VALIDATOR_STAKE_MINIMUM: string;
      VALIDATOR_STAKE_REGISTER: string;
      VALIDATOR_PUNISH_AMOUNT: string;
      VALIDATOR_PUNISH_THRESHOLD: string;
    }
  >requireEnvsSet("OWNER_PRIVATEKEY", "LUMERIN_TOKEN_ADDRESS", "VALIDATOR_STAKE_MINIMUM", "VALIDATOR_STAKE_REGISTER", "VALIDATOR_PUNISH_AMOUNT", "VALIDATOR_PUNISH_THRESHOLD");

  const SAFE_OWNER_ADDRESS = process.env.SAFE_OWNER_ADDRESS as `0x${string}` | undefined;

  const [deployer] = await viem.getWalletClients();
  console.log("Deployer:", deployer.account.address);
  console.log("Safe owner address:", SAFE_OWNER_ADDRESS);

  console.log();

  console.log("Deploying ValidatorRegistry implementation...");
  const impl = await viem.deployContract("ValidatorRegistry");
  console.log("Deployed at:", impl.address);
  await verifyContract(impl.address, []);

  console.log();

  // Checking token
  const token = await viem.getContractAt(
    "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol:IERC20Metadata",
    env.LUMERIN_TOKEN_ADDRESS
  );
  console.log("Token name:", await token.read.name());
  console.log("Token symbol:", await token.read.symbol());
  console.log("Token decimals:", await token.read.decimals());

  // configuting implementation
  console.log("Configuring implementation...");
  console.log("Stake minimum:", env.VALIDATOR_STAKE_MINIMUM);
  console.log("Stake register:", env.VALIDATOR_STAKE_REGISTER);
  console.log("Punish amount:", env.VALIDATOR_PUNISH_AMOUNT);
  console.log("Punish threshold:", env.VALIDATOR_PUNISH_THRESHOLD);
  console.log();

  // Deploy ERC1967Proxy
  console.log("Deploying Proxy...");
  const encodedInitFn = encodeFunctionData({
    abi: impl.abi,
    functionName: "initialize",
    args: [
      env.LUMERIN_TOKEN_ADDRESS,
      BigInt(env.VALIDATOR_STAKE_MINIMUM),
      BigInt(env.VALIDATOR_STAKE_REGISTER),
      BigInt(env.VALIDATOR_PUNISH_AMOUNT),
      Number(env.VALIDATOR_PUNISH_THRESHOLD),
    ],
  });

  const proxy = await viem.deployContract(
    "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy",
    [impl.address, encodedInitFn]
  );
  console.log("Deployed at:", proxy.address);
  // await verifyContract(proxy.address, [impl.address, encodedInitFn]);
  // Get the proxy contract instance
  const validatorRegistry = await viem.getContractAt("ValidatorRegistry", proxy.address);
  console.log("Version:", await validatorRegistry.read.VERSION());

  console.log();

  // Transfer ownership to the owner address
  if (SAFE_OWNER_ADDRESS) {
    console.log("Transfering ownership to:", SAFE_OWNER_ADDRESS);
    const sim = await validatorRegistry.simulate.transferOwnership([SAFE_OWNER_ADDRESS], {
      account: deployer.account.address,
    });
    const receipt = await writeAndWait(deployer, sim);
    console.log("Transaction hash:", receipt.transactionHash);
  }

  console.log("Done!");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
