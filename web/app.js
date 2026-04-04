function switchTab(name) {
	document.querySelectorAll(".tab").forEach(t => t.classList.remove("active"));
	document.querySelectorAll(".tab-content").forEach(t => t.classList.remove("active"));
	document.getElementById("tab-" + name).classList.add("active");
	event.target.classList.add("active");
}

const configFields = [
	{
		key: "ContinuousActiveLimit", label: "ContinuousActiveLimit",
		tip: "How long you can work before getting a break reminder. Focus drops and strain builds after 30m to 1h."
	},
	{
		key: "IdleDurationToConsiderBreak", label: "IdleDurationToConsiderBreak",
		tip: "How long you must be idle for it to count as a break. Even 2m to 5m of rest reduces eye strain and muscle tension."
	},
	{
		key: "NotificationInitialBackoff", label: "NotificationInitialBackoff",
		tip: "Delay before the second reminder if you keep working. Doubles each time (5m, 10m, 20m). Persistent nudges help because we tend to dismiss the first one."
	},
	{
		key: "WebUIPort", label: "WebUIPort",
		tip: "Port number for this web page. Requires restart to take effect."
	},
];

let currentRangeMinutes = parseInt(localStorage.getItem("rangeMinutes")) || 720;
const pollIntervalSec = 10;

function toVN(d) {
	const offset = 7 * 60;
	const vn = new Date(d.getTime() + (offset + d.getTimezoneOffset()) * 60000);
	const pad = n => String(n).padStart(2, "0");
	return vn.getFullYear() + "-" + pad(vn.getMonth() + 1) + "-" + pad(vn.getDate())
		+ "T" + pad(vn.getHours()) + ":" + pad(vn.getMinutes()) + ":" + pad(vn.getSeconds()) + "+07:00";
}

function quickRange(minutes) {
	currentRangeMinutes = minutes;
	localStorage.setItem("rangeMinutes", minutes);
	document.querySelectorAll(".quick-ranges button").forEach(b => b.classList.remove("active"));
	event.target.classList.add("active");
	loadChart();
}

async function loadChart() {
	const now = new Date();
	let url;
	let label;
	let startMs;
	if (currentRangeMinutes === 0) {
		const allStart = "2000-01-01T00:00:00+07:00";
		const endStr = toVN(now);
		url = "/api/history?start=" + encodeURIComponent(allStart) + "&end=" + encodeURIComponent(endStr);
		label = "All time";
		startMs = 0;
	} else {
		const start = new Date(now.getTime() - currentRangeMinutes * 60000);
		startMs = start.getTime();
		const startStr = toVN(start);
		const endStr = toVN(now);
		url = "/api/history?start=" + encodeURIComponent(startStr) + "&end=" + encodeURIComponent(endStr);
		label = "Last " + formatRangeLabel(currentRangeMinutes);
	}
	const resp = await fetch(url);
	const entries = await resp.json();
	if (currentRangeMinutes === 0 && entries.length > 0) {
		startMs = new Date(entries[0].Time).getTime();
	}
	renderChart(entries, label, startMs, now.getTime());
}

// formatDuration, parseDurationToSec, formatRangeLabel, alignedStart are in chart_calc.js

let lastChartState = null;

function renderChart(entries, label, rangeStartMs, rangeEndMs) {
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
		ctx.fillStyle = "#93a1a1";
		ctx.font = "14px system-ui";
		ctx.textAlign = "center";
		ctx.fillText("No data for " + label, W / 2, H / 2);
		document.getElementById("summary").textContent = "";
		lastChartState = null;
		return;
	}

	const spanMs = rangeEndMs - rangeStartMs;
	const spanHours = spanMs / 3600000;

	let bucketMs;
	if (spanHours <= 1) {
		bucketMs = 60000;
	}// 1m bars for 1h (60 bars)
	else if (spanHours <= 4) {
		bucketMs = 5 * 60000;
	}// 5m bars for 4h (48 bars)
	else if (spanHours <= 8) {
		bucketMs = 10 * 60000;
	}// 10m bars for 8h (48 bars)
	else if (spanHours <= 12) {
		bucketMs = 15 * 60000;
	}// 15m bars for 12h (48 bars)
	else if (spanHours <= 24) {
		bucketMs = 30 * 60000;
	}// 30m bars for 24h (48 bars)
	else if (spanHours <= 48) {
		bucketMs = 3600000;
	}// 1h bars for 2d (48 bars)
	else if (spanHours <= 168) {
		bucketMs = 4 * 3600000;
	}// 4h bars for 7d (42 bars)
	else if (spanHours <= 720) {
		bucketMs = 12 * 3600000;
	}// 12h bars for 30d (60 bars)
	else if (spanHours <= 4380) {
		bucketMs = 3 * 86400000;
	}// 3d bars for 6m (60 bars)
	else {
		bucketMs = 7 * 86400000;
	}                        // 7d bars for 1y (52 bars)

	const vnOffsetMs = 7 * 60 * 60000;
	const gridStart = alignedStart(rangeStartMs, bucketMs, vnOffsetMs);

	const pollIntervalMs = pollIntervalSec * 1000;
	const buckets = bucketizeEntries(entries, gridStart, bucketMs, rangeEndMs, pollIntervalMs);

	const totalBuckets = Math.ceil((rangeEndMs - gridStart) / bucketMs);
	const leftPad = 50;
	const rightPad = 10;
	const topPad = 10;
	const bottomPad = 30;
	const barW = Math.max(1, (W - leftPad - rightPad) / totalBuckets);
	const chartH = H - topPad - bottomPad;

	for (let i = 0; i < totalBuckets; i++) {
		const b = buckets[i];
		if (!b) {
			continue;
		}
		const bucketSec = bucketMs / 1000;
		const activeH = (b.activeSec / bucketSec) * chartH;
		const x = leftPad + i * barW;

		ctx.fillStyle = "#859900";
		ctx.fillRect(x, topPad + chartH - activeH, barW - 1, activeH);
		ctx.fillStyle = "#d3cbb7";
		ctx.fillRect(x, topPad, barW - 1, chartH - activeH);
	}

	ctx.fillStyle = "#93a1a1";
	ctx.font = "11px system-ui";
	ctx.textAlign = "center";
	const pad2 = n => String(n).padStart(2, "0");

	const labelCount = 6;
	const labelStep = Math.max(1, Math.ceil(totalBuckets / labelCount));
	for (let i = 0; i <= totalBuckets; i += labelStep) {
		const t = new Date(gridStart + i * bucketMs + vnOffsetMs);
		let text;
		if (spanHours <= 1) {
			text = pad2(t.getUTCHours()) + ":" + pad2(t.getUTCMinutes());
		} else if (spanHours <= 48) {
			text = pad2(t.getUTCHours()) + ":" + pad2(t.getUTCMinutes());
		} else {
			text = t.getUTCFullYear() + "-" + pad2(t.getUTCMonth() + 1) + "-" + pad2(t.getUTCDate());
		}
		const x = leftPad + i * barW;
		ctx.fillText(text, x, H - 6);
	}

	const {activeSec, totalSec} = calcActiveDuration(entries, rangeStartMs, rangeEndMs, pollIntervalMs);
	const pct = totalSec > 0 ? Math.round(activeSec / totalSec * 100) : 0;
	document.getElementById("summary").textContent =
		label + " | Active: " + formatDuration(activeSec) + " (" + pct + "%)";

	lastChartState = {
		buckets, totalBuckets, gridStart, bucketMs, vnOffsetMs,
		leftPad, topPad, bottomPad, barW, chartH, W, H, spanHours
	};
}

document.getElementById("canvas").addEventListener("mousemove", function (e) {
	const tooltip = document.getElementById("tooltip");
	if (!lastChartState) {
		tooltip.style.display = "none";
		return;
	}
	const s = lastChartState;
	const rect = e.target.getBoundingClientRect();
	const x = e.clientX - rect.left;
	const y = e.clientY - rect.top;

	if (y < s.topPad || y > s.H - s.bottomPad || x < s.leftPad) {
		tooltip.style.display = "none";
		return;
	}

	const bucketIdx = Math.floor((x - s.leftPad) / s.barW);
	const b = s.buckets[bucketIdx];
	if (!b) {
		tooltip.style.display = "none";
		return;
	}

	const pad2 = n => String(n).padStart(2, "0");
	const bucketTime = new Date(s.gridStart + bucketIdx * s.bucketMs + s.vnOffsetMs);
	const dateStr = bucketTime.getUTCFullYear() + "-" + pad2(bucketTime.getUTCMonth() + 1) + "-" + pad2(bucketTime.getUTCDate());
	let timeStr;
	if (s.spanHours <= 48) {
		timeStr = dateStr + " " + pad2(bucketTime.getUTCHours()) + ":" + pad2(bucketTime.getUTCMinutes());
	} else {
		timeStr = dateStr;
	}

	const bucketSec = s.bucketMs / 1000;
	const activeDur = formatDuration(Math.round(b.activeSec));
	const totalDur = formatDuration(Math.round(bucketSec));
	const pct = Math.round(b.activeSec / bucketSec * 100);
	tooltip.textContent = timeStr + " | Active: " + activeDur + " / " + totalDur + " (" + pct + "%)";
	tooltip.style.display = "block";
	let left = e.clientX - rect.left + 12;
	const tooltipWidth = tooltip.offsetWidth;
	if (left + tooltipWidth > rect.width) {
		left = e.clientX - rect.left - tooltipWidth - 8;
	}
	tooltip.style.left = left + "px";
	tooltip.style.top = (e.clientY - rect.top - 8) + "px";
});

document.getElementById("canvas").addEventListener("mouseleave", function () {
	document.getElementById("tooltip").style.display = "none";
});

async function loadConfig() {
	const resp = await fetch("/api/config");
	const cfg = await resp.json();
	const section = document.getElementById("configSection");
	section.innerHTML = "";
	for (const f of configFields) {
		const div = document.createElement("div");
		div.className = "config-field";
		const labelWrap = document.createElement("div");
		const label = document.createElement("label");
		label.textContent = f.label;
		label.title = f.tip;
		labelWrap.appendChild(label);
		const tip = document.createElement("div");
		tip.className = "config-tip";
		tip.textContent = f.tip;
		labelWrap.appendChild(tip);
		const input = document.createElement("input");
		input.id = "cfg-" + f.key;
		input.value = cfg[f.key] || "";
		div.appendChild(labelWrap);
		div.appendChild(input);
		section.appendChild(div);
	}
	const btn = document.createElement("button");
	btn.className = "save-btn";
	btn.textContent = "Save config";
	btn.onclick = saveConfig;
	section.appendChild(btn);
}

async function saveConfig() {
	const body = {};
	for (const f of configFields) {
		const val = document.getElementById("cfg-" + f.key).value;
		if (f.key === "WebUIPort") {
			body[f.key] = parseInt(val, 10);
		} else {
			body[f.key] = val;
		}
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

// Highlight the saved range button on load.
document.querySelectorAll(".quick-ranges button").forEach(b => {
	if (parseInt(b.dataset.range) === currentRangeMinutes) {
		b.classList.add("active");
	}
});

async function testNotification() {
	const resp = await fetch("/api/test-notification", {method: "POST"});
	const result = await resp.json();
	const el = document.getElementById("testResult");
	el.textContent = "Sent! Active: " + result.activeDuration;
	setTimeout(() => el.textContent = "", 3000);
}

async function loadAutostart() {
	const resp = await fetch("/api/autostart");
	const data = await resp.json();
	document.getElementById("autostartCheckbox").checked = data.IsEnabled;
}

async function toggleAutostart() {
	const isEnabled = document.getElementById("autostartCheckbox").checked;
	const resp = await fetch("/api/autostart", {
		method: "POST",
		headers: {"Content-Type": "application/json"},
		body: JSON.stringify({IsEnabled: isEnabled}),
	});
	if (!resp.ok) {
		const text = await resp.text();
		alert("Failed to update autostart: " + text);
		document.getElementById("autostartCheckbox").checked = !isEnabled;
	}
}

loadChart();
loadConfig();
loadAutostart();
setInterval(loadChart, 60000);
