import { useState } from "react";
import Button from "../Button";
import Panel from "../Panel";

export default function PendingBattle({ inviteCode }: { inviteCode: string }) {
  const [copyStatus, setCopyStatus] = useState("");

  async function copyInviteCode() {
    try {
      await navigator.clipboard.writeText(inviteCode);
      setCopyStatus("Invite code copied.");
    } catch {
      setCopyStatus("Could not copy automatically. Select and copy the code below.");
    }
  }

  return (
    <Panel className="text-center">
      <h2 className="text-2xl font-semibold">Waiting for an opponent</h2>
      <p className="my-4 text-muted">Share this invite code.</p>
      <p className="select-all font-mono text-3xl tracking-widest">{inviteCode}</p>

      <Button className="mt-5" onClick={() => void copyInviteCode()}>
        Copy code
      </Button>

      <p role="status" className="mt-3 min-h-6 text-sm text-muted">
        {copyStatus}
      </p>
    </Panel>
  );
}
