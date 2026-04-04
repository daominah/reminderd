function formatDuration(totalSeconds) {
	const h = Math.floor(totalSeconds / 3600);
	const m = Math.floor((totalSeconds % 3600) / 60);
	if (h > 0 && m > 0) {
		return h + "h" + m + "m";
	}
	if (h > 0) {
		return h + "h";
	}
	return m + "m";
}

function parseDurationToSec(s) {
	let total = 0;
	const hMatch = s.match(/(\d+)h/);
	const mMatch = s.match(/(\d+)m/);
	const sMatch = s.match(/(\d+)s/);
	if (hMatch) {
		total += parseInt(hMatch[1]) * 3600;
	}
	if (mMatch) {
		total += parseInt(mMatch[1]) * 60;
	}
	if (sMatch) {
		total += parseInt(sMatch[1]);
	}
	return total;
}

function formatRangeLabel(minutes) {
	if (minutes < 60) {
		return minutes + "m";
	}
	if (minutes < 1440) {
		return (minutes / 60) + "h";
	}
	if (minutes < 43200) {
		return (minutes / 1440) + "d";
	}
	if (minutes < 525600) {
		return (minutes / 43200) + "mo";
	}
	return (minutes / 525600) + "y";
}

function alignedStart(rangeStartMs, bucketMs, vnOffsetMs) {
	return Math.floor((rangeStartMs + vnOffsetMs) / bucketMs) * bucketMs - vnOffsetMs;
}

// entrySpan returns [startMs, endMs) for an entry.
// Raw entry covers [Time, Time + pollIntervalMs).
// Compact entry covers [Time, TimeCompactEnd + pollIntervalMs).
function entrySpan(e, pollIntervalMs) {
	const start = new Date(e.Time).getTime();
	let end;
	if (e.IsCompact && e.TimeCompactEnd) {
		end = new Date(e.TimeCompactEnd).getTime() + pollIntervalMs;
	} else {
		end = start + pollIntervalMs;
	}
	return {start, end};
}

// calcActiveDuration computes the total active seconds from history entries.
// Gaps between entries are treated as IDLE.
function calcActiveDuration(entries, rangeStartMs, rangeEndMs, pollIntervalMs) {
	if (entries.length === 0) {
		return {activeSec: 0, totalSec: 0};
	}
	let activeSec = 0;
	for (let i = 0; i < entries.length; i++) {
		const span = entrySpan(entries[i], pollIntervalMs);
		const spanEnd = Math.min(span.end, rangeEndMs);
		const durationSec = Math.max(0, (spanEnd - span.start) / 1000);
		if (entries[i].State === "ACTIVE") {
			activeSec += durationSec;
		}
	}
	const totalSec = (rangeEndMs - rangeStartMs) / 1000;
	return {activeSec: Math.round(activeSec), totalSec: Math.round(totalSec)};
}

// bucketizeEntries distributes each entry's time span into buckets.
// Gaps between entries are not distributed (treated as empty time).
// Returns an object keyed by bucket index, each value has {activeSec, idleSec}.
function bucketizeEntries(entries, gridStartMs, bucketMs, rangeEndMs, pollIntervalMs) {
	const buckets = {};
	for (let i = 0; i < entries.length; i++) {
		const span = entrySpan(entries[i], pollIntervalMs);
		const spanEnd = Math.min(span.end, rangeEndMs);
		const state = entries[i].State;

		const firstBucket = Math.floor((span.start - gridStartMs) / bucketMs);
		const lastBucket = Math.floor((Math.max(span.start, spanEnd - 1) - gridStartMs) / bucketMs);

		for (let b = firstBucket; b <= lastBucket; b++) {
			const bucketStartMs = gridStartMs + b * bucketMs;
			const bucketEndMs = bucketStartMs + bucketMs;
			const overlapStart = Math.max(span.start, bucketStartMs);
			const overlapEnd = Math.min(spanEnd, bucketEndMs);
			const overlapSec = (overlapEnd - overlapStart) / 1000;
			if (overlapSec <= 0) {
				continue;
			}

			if (!buckets[b]) {
				buckets[b] = {activeSec: 0, idleSec: 0};
			}
			if (state === "ACTIVE") {
				buckets[b].activeSec += overlapSec;
			} else {
				buckets[b].idleSec += overlapSec;
			}
		}
	}
	return buckets;
}

// Export for Node.js tests, no-op in browser.
if (typeof module !== "undefined") {
	module.exports = {formatDuration, parseDurationToSec, formatRangeLabel, alignedStart, calcActiveDuration, bucketizeEntries};
}
