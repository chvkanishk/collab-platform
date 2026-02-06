export async function fetchHistory(channelId) {
  try {
    const res = await fetch(`http://localhost:8084/history?channelId=${channelId}`);
    if (!res.ok) {
      console.error("Failed to fetch history");
      return [];
    }
    return await res.json();
  } catch (err) {
    console.error("Error fetching history:", err);
    return [];
  }
}
