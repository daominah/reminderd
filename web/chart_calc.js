function formatDuration(totalSeconds) {
  const h = Math.floor(totalSeconds / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  if (h > 0 && m > 0) return h + "h" + m + "m";
  if (h > 0) return h + "h";
  return m + "m";
}

function parseDurationToSec(s) {
  let total = 0;
  const hMatch = s.match(/(\d+)h/);
  const mMatch = s.match(/(\d+)m/);
  const sMatch = s.match(/(\d+)s/);
  if (hMatch) total += parseInt(hMatch[1]) * 3600;
  if (mMatch) total += parseInt(mMatch[1]) * 60;
  if (sMatch) total += parseInt(sMatch[1]);
  return total;
}

function formatRangeLabel(minutes) {
  if (minutes < 60) return minutes + "m";
  if (minutes < 1440) return (minutes / 60) + "h";
  if (minutes < 43200) return (minutes / 1440) + "d";
  if (minutes < 525600) return (minutes / 43200) + "mo";
  return (minutes / 525600) + "y";
}

function alignedStart(rangeStartMs, bucketMs, vnOffsetMs) {
  return Math.floor((rangeStartMs + vnOffsetMs) / bucketMs) * bucketMs - vnOffsetMs;
}

// calcActiveDuration computes the total active seconds from history entries.
// Each entry represents a state boundary: "this state started at this time."
// An entry's duration lasts until the next entry (or rangeEndMs for the last one).
function calcActiveDuration(entries, rangeEndMs) {
  if (entries.length === 0) return { activeSec: 0, totalSec: 0 };
  let activeSec = 0;
  const startMs = new Date(entries[0].Time).getTime();
  for (let i = 0; i < entries.length; i++) {
    const t = new Date(entries[i].Time).getTime();
    const next = (i + 1 < entries.length) ? new Date(entries[i + 1].Time).getTime() : rangeEndMs;
    const clampedNext = Math.min(next, rangeEndMs);
    const durationSec = Math.max(0, (clampedNext - t) / 1000);
    if (entries[i].State === "ACTIVE") {
      activeSec += durationSec;
    }
  }
  const totalSec = (rangeEndMs - startMs) / 1000;
  return { activeSec: Math.round(activeSec), totalSec: Math.round(totalSec) };
}

// bucketizeEntries distributes each entry's time span into buckets.
// Each entry's state lasts until the next entry (or rangeEndMs for the last).
// Returns an object keyed by bucket index, each value has {activeSec, idleSec}.
function bucketizeEntries(entries, gridStartMs, bucketMs, rangeEndMs) {
  const buckets = {};
  for (let i = 0; i < entries.length; i++) {
    const spanStart = new Date(entries[i].Time).getTime();
    const spanEnd = (i + 1 < entries.length) ? new Date(entries[i + 1].Time).getTime() : rangeEndMs;
    const state = entries[i].State;

    const firstBucket = Math.floor((spanStart - gridStartMs) / bucketMs);
    const lastBucket = Math.floor((spanEnd - 1 - gridStartMs) / bucketMs);

    for (let b = firstBucket; b <= lastBucket; b++) {
      const bucketStartMs = gridStartMs + b * bucketMs;
      const bucketEndMs = bucketStartMs + bucketMs;
      const overlapStart = Math.max(spanStart, bucketStartMs);
      const overlapEnd = Math.min(spanEnd, bucketEndMs);
      const overlapSec = (overlapEnd - overlapStart) / 1000;
      if (overlapSec <= 0) continue;

      if (!buckets[b]) buckets[b] = {activeSec: 0, idleSec: 0};
      if (state === "ACTIVE") buckets[b].activeSec += overlapSec;
      else buckets[b].idleSec += overlapSec;
    }
  }
  return buckets;
}

// Export for Node.js tests, no-op in browser.
if (typeof module !== "undefined") {
  module.exports = { formatDuration, parseDurationToSec, formatRangeLabel, alignedStart, calcActiveDuration, bucketizeEntries };
}
