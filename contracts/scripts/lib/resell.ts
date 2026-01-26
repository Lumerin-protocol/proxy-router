import { viem } from "hardhat";
import { mapResellTerms } from "../../tests/mappers";

export async function getResellChain(contractAddress: `0x${string}`, index: number) {
  const implementation = await viem.getContractAt("Implementation", contractAddress);
  const data = await implementation.read.resellChain([BigInt(index)]);
  return mapResellTerms(data);
}

type ResellTerms = Awaited<ReturnType<typeof getResellChain>>;

export async function getFullResellChain(contractAddress: `0x${string}`) {
  const data: ResellTerms[] = [];
  for (let i = 0; ; i++) {
    try {
      const d = await getResellChain(contractAddress, i);
      data.push(d);
    } catch (e) {
      break;
    }
  }
  return data;
}
