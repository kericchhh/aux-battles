import { useState } from "react";
import { API_URL } from "../../api/client";
import type { BattleRound } from "../../api/battles";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";

export default function AudioPlayer({ round }: { round: BattleRound }) {
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);

  function retryAudio() {
    setFailed(false);
    setRetry((current) => current + 1);
  }

  return (
    <div>
      <audio
        key={retry}
        controls
        preload="none"
        crossOrigin="use-credentials"
        className="w-full"
        onError={() => setFailed(true)}
        onLoadedData={() => setFailed(false)}
        src={`${API_URL}/rounds/${round.id}/audio?stage=${round.myStage}&retry=${retry}`}
      />

      {failed && (
        <div className="mt-3 space-y-3">
          <ErrorBanner message="Audio could not be loaded. Your session may have expired or the file may be unavailable." />
          <Button variant="secondary" onClick={retryAudio}>
            Retry audio
          </Button>
        </div>
      )}
    </div>
  );
}
