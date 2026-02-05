export default function Sidebar({ teamId, setChannelId }) {
  // For now, channels are static.
  // Later we can load them dynamically from your backend.
  const channels = [
    { id: 1, name: "general" },
    { id: 2, name: "engineering" },
    { id: 3, name: "random" },
  ];

  return (
    <div
      style={{
        width: 220,
        background: "#1e1e1e",
        color: "white",
        padding: 20,
        display: "flex",
        flexDirection: "column",
        gap: 10,
      }}
    >
      <h3 style={{ marginBottom: 10 }}>Channels</h3>

      {channels.map((c) => (
        <div
          key={c.id}
          style={{
            padding: "8px 12px",
            cursor: "pointer",
            borderRadius: 4,
            background: "#2a2a2a",
          }}
          onClick={() => setChannelId(c.id)}
        >
          #{c.name}
        </div>
      ))}
    </div>
  );
}
