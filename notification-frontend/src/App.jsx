import { useState } from "react";
import Sidebar from "./components/Sidebar";
import Chat from "./components/Chat";

function App() {
  const [teamId] = useState(1);
  const [channelId, setChannelId] = useState(1);

  return (
    <div style={{ display: "flex", height: "100vh" }}>
      <Sidebar teamId={teamId} setChannelId={setChannelId} />
      <Chat channelId={channelId} />
    </div>
  );
}

export default App;
