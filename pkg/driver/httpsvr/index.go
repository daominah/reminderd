package httpsvr

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>reminderd</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: system-ui, sans-serif; background: #1a1a2e; color: #e0e0e0; padding: 24px; }
  h1 { font-size: 1.4em; margin-bottom: 16px; }
  h2 { font-size: 1.1em; margin: 24px 0 12px; }
  .controls { display: flex; gap: 12px; align-items: center; margin-bottom: 20px; flex-wrap: wrap; }
  .controls label { font-size: 0.9em; }
  .controls input[type="date"] { background: #16213e; color: #e0e0e0; border: 1px solid #0f3460; padding: 4px 8px; border-radius: 4px; }
  .controls button { background: #0f3460; color: #e0e0e0; border: none; padding: 6px 16px; border-radius: 4px; cursor: pointer; }
  .controls button:hover { background: #533483; }
  #chart { width: 100%; height: 300px; background: #16213e; border-radius: 8px; overflow: hidden; }
  canvas { width: 100%; height: 100%; }
  .config-section { max-width: 500px; }
  .config-field { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid #0f3460; }
  .config-field label { font-size: 0.9em; }
  .config-field input { background: #16213e; color: #e0e0e0; border: 1px solid #0f3460; padding: 4px 8px; border-radius: 4px; width: 120px; text-align: right; }
  .save-btn { margin-top: 12px; }
  .status { font-size: 0.85em; color: #6c6; margin-left: 12px; }
  .summary { font-size: 0.9em; color: #aaa; margin: 8px 0; }
</style>
</head>
<body>
<h1>reminderd: activity history</h1>

<div class="controls">
  <label>Date: <input type="date" id="dateInput"></label>
  <button onclick="loadChart()">Load</button>
  <button onclick="loadToday()">Today</button>
</div>
<div id="summary" class="summary"></div>
<div id="chart"><canvas id="canvas"></canvas></div>

<h2>Configuration</h2>
<div class="config-section" id="configSection"></div>

<script>
const configFields = [
  {key: "ContinuousActiveLimit", label: "Active limit"},
  {key: "IdleDurationToConsiderBreak", label: "Idle threshold"},
  {key: "KeyboardMouseInputPollInterval", label: "Poll interval"},
  {key: "NotificationInitialBackoff", label: "Initial backoff"},
  {key: "WebUIPort", label: "Web UI port"},
];

function todayStr() {
  const d = new Date();
  const offset = 7 * 60;
  const local = new Date(d.getTime() + (offset + d.getTimezoneOffset()) * 60000);
  return local.toISOString().slice(0, 10);
}

function loadToday() {
  document.getElementById("dateInput").value = todayStr();
  loadChart();
}

async function loadChart() {
  const date = document.getElementById("dateInput").value;
  if (!date) return;
  const start = date + "T00:00:00+07:00";
  const end = date + "T23:59:59+07:00";
  const url = "/api/history?start=" + encodeURIComponent(start) + "&end=" + encodeURIComponent(end);
  const resp = await fetch(url);
  const entries = await resp.json();
  renderChart(entries, date);
}

function renderChart(entries, date) {
  const canvas = document.getElementById("canvas");
  const ctx = canvas.getContext("2d");
  const rect = canvas.parentElement.getBoundingClientRect();
  canvas.width = rect.width * devicePixelRatio;
  canvas.height = rect.height * devicePixelRatio;
  ctx.scale(devicePixelRatio, devicePixelRatio);
  const W = rect.width;
  const H = rect.height;
  ctx.clearRect(0, 0, W, H);

  if (entries.length === 0) {
    ctx.fillStyle = "#666";
    ctx.font = "14px system-ui";
    ctx.textAlign = "center";
    ctx.fillText("No data for " + date, W / 2, H / 2);
    document.getElementById("summary").textContent = "";
    return;
  }

  // Determine bucket size based on data span.
  const times = entries.map(e => new Date(e.Time).getTime());
  const minT = Math.min(...times);
  const maxT = Math.max(...times);
  const spanMs = maxT - minT;
  const spanHours = spanMs / 3600000;

  let bucketMs;
  if (spanHours > 12) bucketMs = 3600000;
  else if (spanHours > 3) bucketMs = 600000;
  else bucketMs = 60000;

  // Use full day range for x-axis.
  const dayStart = new Date(date + "T00:00:00+07:00").getTime();
  const dayEnd = new Date(date + "T23:59:59+07:00").getTime();

  const buckets = {};
  for (const e of entries) {
    const t = new Date(e.Time).getTime();
    const bucket = Math.floor((t - dayStart) / bucketMs);
    if (!buckets[bucket]) buckets[bucket] = {active: 0, idle: 0};
    if (e.State === "ACTIVE") buckets[bucket].active++;
    else buckets[bucket].idle++;
  }

  const totalBuckets = Math.ceil((dayEnd - dayStart) / bucketMs);
  const barW = Math.max(1, (W - 60) / totalBuckets);
  const leftPad = 40;
  const topPad = 10;
  const bottomPad = 30;
  const chartH = H - topPad - bottomPad;

  // Draw bars.
  let totalActive = 0;
  for (let i = 0; i < totalBuckets; i++) {
    const b = buckets[i];
    if (!b) continue;
    const total = b.active + b.idle;
    totalActive += b.active;
    const activeH = (b.active / total) * chartH;
    const x = leftPad + i * barW;

    ctx.fillStyle = "#2ecc71";
    ctx.fillRect(x, topPad + chartH - activeH, barW - 1, activeH);
    ctx.fillStyle = "#555";
    ctx.fillRect(x, topPad, barW - 1, chartH - activeH);
  }

  // Draw x-axis labels.
  ctx.fillStyle = "#888";
  ctx.font = "11px system-ui";
  ctx.textAlign = "center";
  for (let h = 0; h <= 24; h += (spanHours > 12 ? 3 : 1)) {
    const x = leftPad + (h * 3600000 / bucketMs) * barW;
    ctx.fillText(h + ":00", x, H - 6);
  }

  // Summary.
  const totalEntries = entries.length;
  const activeEntries = entries.filter(e => e.State === "ACTIVE").length;
  const pollSec = bucketMs === 60000 ? 60 : (bucketMs === 600000 ? 600 : 3600);
  document.getElementById("summary").textContent =
    "Active: " + activeEntries + " / " + totalEntries + " samples";
}

async function loadConfig() {
  const resp = await fetch("/api/config");
  const cfg = await resp.json();
  const section = document.getElementById("configSection");
  section.innerHTML = "";
  for (const f of configFields) {
    const div = document.createElement("div");
    div.className = "config-field";
    const label = document.createElement("label");
    label.textContent = f.label;
    const input = document.createElement("input");
    input.id = "cfg-" + f.key;
    input.value = cfg[f.key] || "";
    div.appendChild(label);
    div.appendChild(input);
    section.appendChild(div);
  }
  const btn = document.createElement("button");
  btn.className = "controls save-btn";
  btn.textContent = "Save config";
  btn.onclick = saveConfig;
  section.appendChild(btn);
}

async function saveConfig() {
  const body = {};
  for (const f of configFields) {
    const val = document.getElementById("cfg-" + f.key).value;
    if (f.key === "WebUIPort") body[f.key] = parseInt(val, 10);
    else body[f.key] = val;
  }
  const resp = await fetch("/api/config", {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify(body),
  });
  if (resp.ok) {
    const status = document.createElement("span");
    status.className = "status";
    status.textContent = "Saved!";
    document.querySelector(".save-btn").after(status);
    setTimeout(() => status.remove(), 2000);
  }
}

document.getElementById("dateInput").value = todayStr();
loadChart();
loadConfig();
</script>
</body>
</html>`
