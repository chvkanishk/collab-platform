import { useEffect, useRef, useState } from "react";
import { fetchHistory } from "../api/history";

export default function useWebSocket(channelId) {
  const [messages, setMessages] = useState([]);
  const socketRef = useRef(null);

  // Load history when channel changes
  useEffect(() => {
    if (!channelId) return;

    fetchHistory(channelId).then((history) => {
      setMessages(history);
    });
  }, [channelId]);

  // WebSocket connection
  useEffect(() => {
    if (!channelId) return;

    const ws = new WebSocket(`ws://localhost:8083/ws?channelId=${channelId}`);
    socketRef.current = ws;

    ws.onopen = () => {
      console.log("WebSocket connected");
    };

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      setMessages((prev) => [...prev, msg]);
    };

    ws.onclose = () => {
      console.log("WebSocket closed");
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    return () => {
      ws.close();
    };
  }, [channelId]);

  const sendMessage = (text) => {
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket not connected");
      return;
    }

    const payload = {
      channelId,
      text,
      sender: "You",
      timestamp: Date.now(),
    };

    socketRef.current.send(JSON.stringify(payload));
  };

  return { messages, sendMessage };
}
