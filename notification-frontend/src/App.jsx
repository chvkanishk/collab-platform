import { useState } from "react";
import Sidebar from "./components/Sidebar";

function App() {
  const [teamId] = useState(1);
  const [channelId, setChannelId] = useState(1);

  return (
    <div style={{ display: "flex", height: "100vh" }}>
      <Sidebar teamId={teamId} setChannelId={setChannelId} />

      <div style={{ flex: 1, padding: 40 }}>
        <h1>Channel {channelId}</h1>
        <p>Chat UI coming next…</p>
      </div>
    </div>
  );
}

export default App;
