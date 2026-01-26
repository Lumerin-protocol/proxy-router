import type { Account, Chain, PublicClient, Transport, WalletClient } from "viem";
import { SafeWallet } from "./safe";
import { waitForTransactionReceipt } from "viem/actions";

type TransactionData = {
  to: `0x${string}`;
  value?: bigint;
  data?: `0x${string}`;
};

type ProposerProps = {
  signer: WalletClient<Transport, Chain, Account>;
  safeAddress?: `0x${string}`;
};

export class Proposer {
  private readonly props: ProposerProps;
  private readonly safe?: SafeWallet;

  constructor(props: ProposerProps) {
    this.props = props;
    if (props.safeAddress) {
      this.safe = new SafeWallet(props.safeAddress, props.signer);
    }
  }

  async propose(tx: TransactionData): Promise<string> {
    const pc: PublicClient = this.props.signer;

    // simulate the call to check if it's valid
    await pc.call(tx);

    if (!this.safe) {
      const hash = await this.props.signer.sendTransaction(tx);
      const receipt = await waitForTransactionReceipt(this.props.signer, {
        hash,
      });
      console.log("Transaction submitted on chain:\n", receipt.transactionHash);
      return receipt.transactionHash as string;
    }

    const safeTx = await this.safe.proposeTransaction({
      data: tx.data || "0x",
      to: tx.to,
      value: tx.value?.toString() || "0",
    });
    const safeTxURL = this.safe.getSafeUITxUrl(safeTx);
    console.log("Transaction proposed on Safe:\n", safeTxURL);
    return safeTxURL;
  }
}
