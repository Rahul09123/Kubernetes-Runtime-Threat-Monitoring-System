const alertsBody = document.getElementById("alertsBody");
const apiStatus = document.getElementById("apiStatus");
const lastUpdated = document.getElementById("lastUpdated");
const latestAlert = document.getElementById("latestAlert");
const latestAlertTitle = document.getElementById("latestAlertTitle");
const latestAlertDetails = document.getElementById("latestAlertDetails");
const totalCount = document.getElementById("totalCount");
const criticalCount = document.getElementById("criticalCount");
const highCount = document.getElementById("highCount");
const mediumCount = document.getElementById("mediumCount");
const namespaceCount = document.getElementById("namespaceCount");
const categoryCount = document.getElementById("categoryCount");
const podCount = document.getElementById("podCount");

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

function mostCommon(values) {
  const counts = new Map();
  values.forEach((value) => {
    if (!value) return;
    counts.set(value, (counts.get(value) || 0) + 1);
  });

  let winner = "-";
  let max = 0;
  counts.forEach((count, value) => {
    if (count > max) {
      winner = value;
      max = count;
    }
  });

  return { value: winner, count: max };
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

function renderSnapshot(alerts) {
  const namespaces = mostCommon(alerts.map((alert) => alert.namespace));
  const categories = mostCommon(alerts.map((alert) => alert.category));
  const pods = mostCommon(alerts.map((alert) => alert.pod));
  const latest = alerts[0];

  namespaceCount.textContent = namespaces.count ? `${namespaces.value} (${namespaces.count})` : "-";
  categoryCount.textContent = categories.count ? `${categories.value} (${categories.count})` : "-";
  podCount.textContent = pods.count ? `${pods.value} (${pods.count})` : "-";

  if (latest) {
    latestAlert.textContent = `Latest alert: ${latest.severity || "low"}`;
    latestAlertTitle.textContent = latest.summary || "Untitled alert";
    latestAlertDetails.textContent = `${latest.severity || "low"} / ${latest.category || "unknown"} / ${latest.namespace || "-"} / ${latest.pod || "-"} / ${formatDate(latest.detected)}`;
  } else {
    latestAlert.textContent = "Latest alert: waiting for data";
    latestAlertTitle.textContent = "No alerts yet";
    latestAlertDetails.textContent = "Trigger a test pod or wait for runtime activity to see the pipeline in motion.";
  }
}

async function loadDashboard() {
  try {
    const [healthResponse, alertsResponse] = await Promise.all([
      fetch("/healthz", { cache: "no-store" }),
      fetch("/api/alerts", { cache: "no-store" }),
    ]);

    if (!healthResponse.ok) {
      throw new Error(`health check failed: ${healthResponse.status}`);
    }
    if (!alertsResponse.ok) {
      throw new Error(`alerts fetch failed: ${alertsResponse.status}`);
    }
    const alerts = await alertsResponse.json();
    apiStatus.textContent = "API: healthy";
    apiStatus.className = "status-chip live";
    renderCounters(alerts);
    renderSnapshot(alerts);
    renderAlerts(alerts);
    lastUpdated.textContent = `Last sync: ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    apiStatus.textContent = "API: degraded";
    apiStatus.className = "status-chip warn";
    lastUpdated.textContent = `Dashboard error: ${error.message}`;
  }
}

loadDashboard();
setInterval(loadDashboard, 5000);
