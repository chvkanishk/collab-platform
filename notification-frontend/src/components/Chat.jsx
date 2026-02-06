import { useState } from "react";
import useWebSocket from "../hooks/useWebSocket";

export default function Chat({ channelId }) {
  const { messages, sendMessage } = useWebSocket(channelId);
  const [text, setText] = useState("");

  const handleSend = () => {
    if (!text.trim()) return;
    sendMessage(text);
    setText("");
  };

  return (
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        height: "100%",
        padding: 20,
      }}
    >
      <h2>Channel {channelId}</h2>

      {/* Message list */}
      <div
        style={{
          flex: 1,
          background: "#f5f5f5",
          borderRadius: 8,
          padding: 16,
          overflowY: "auto",
          marginBottom: 16,
        }}
      >
        {messages.length === 0 && <p>No messages yet…</p>}

        {messages.map((msg, i) => (
          <div key={i} style={{ marginBottom: 12 }}>
            <strong>{msg.sender}:</strong> {msg.text}
          </div>
        ))}
      </div>

      {/* Message input */}
      <div style={{ display: "flex", gap: 8 }}>
        <input
          type="text"
          placeholder="Type a message…"
          value={text}
          onChange={(e) => setText(e.target.value)}
          style={{
            flex: 1,
            padding: 12,
            borderRadius: 6,
            border: "1px solid #ccc",
          }}
        />
        <button
          onClick={handleSend}
          style={{
            padding: "12px 20px",
            background: "#007bff",
            color: "white",
            border: "none",
            borderRadius: 6,
            cursor: "pointer",
          }}
        >
          Send
        </button>
      </div>
    </div>
  );
}
