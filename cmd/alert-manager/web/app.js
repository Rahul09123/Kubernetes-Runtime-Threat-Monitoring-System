const alertsBody = document.getElementById("alertsBody");
const lastUpdated = document.getElementById("lastUpdated");
const totalCount = document.getElementById("totalCount");
const criticalCount = document.getElementById("criticalCount");
const highCount = document.getElementById("highCount");
const mediumCount = document.getElementById("mediumCount");

function severityClass(severity) {
  const value = String(severity || "low").toLowerCase();
  if (value === "critical" || value === "high" || value === "medium") {
    return value;
  }
  return "low";
}

function formatDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function renderAlerts(alerts) {
  alertsBody.innerHTML = "";

  if (!alerts.length) {
    const row = document.createElement("tr");
    row.innerHTML = '<td class="empty" colspan="7">No alerts detected yet. Trigger test alerts or wait for runtime events.</td>';
    alertsBody.appendChild(row);
    return;
  }

  alerts.forEach((alert) => {
    const row = document.createElement("tr");
    const sevClass = severityClass(alert.severity);
    row.innerHTML = `
      <td><span class="badge ${sevClass}">${alert.severity || "low"}</span></td>
      <td>${alert.category || "-"}</td>
      <td>${alert.namespace || "-"}</td>
      <td>${alert.pod || "-"}</td>
      <td>${alert.source || "-"}</td>
      <td>${alert.summary || "-"}</td>
      <td>${formatDate(alert.detected)}</td>
    `;
    alertsBody.appendChild(row);
  });
}

function renderCounters(alerts) {
  const counts = alerts.reduce(
    (acc, alert) => {
      const sev = String(alert.severity || "").toLowerCase();
      if (sev === "critical") acc.critical += 1;
      if (sev === "high") acc.high += 1;
      if (sev === "medium") acc.medium += 1;
      return acc;
    },
    { critical: 0, high: 0, medium: 0 },
  );

  totalCount.textContent = String(alerts.length);
  criticalCount.textContent = String(counts.critical);
  highCount.textContent = String(counts.high);
  mediumCount.textContent = String(counts.medium);
}

async function loadAlerts() {
  try {
    const response = await fetch("/api/alerts", { cache: "no-store" });
    if (!response.ok) {
      throw new Error(`fetch failed: ${response.status}`);
    }
    const alerts = await response.json();
    renderCounters(alerts);
    renderAlerts(alerts);
    lastUpdated.textContent = `Last sync: ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    lastUpdated.textContent = `Dashboard error: ${error.message}`;
  }
}

loadAlerts();
setInterval(loadAlerts, 5000);
