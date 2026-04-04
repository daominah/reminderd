// Run with: node web/chart_calc_test.js

const {formatDuration, parseDurationToSec, formatRangeLabel, alignedStart, calcActiveDuration, bucketizeEntries} = require("./chart_calc");

let passed = 0;
let failed = 0;

function assertEqual(actual, expected, name) {
	if (actual === expected) {
		passed++;
	} else {
		failed++;
		console.error(`FAIL ${name}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
	}
}

const pollMs = 10000; // 10s poll interval

function activePct(entries, rangeStartMs, rangeEndMs) {
	const {activeSec, totalSec} = calcActiveDuration(entries, rangeStartMs, rangeEndMs, pollMs);
	return totalSec > 0 ? Math.round(activeSec / totalSec * 100) : 0;
}

// --- formatDuration ---

// WHEN formatting zero seconds
// THEN returns "0m"
assertEqual(formatDuration(0), "0m", "formatDuration(0)");

// WHEN formatting seconds less than a minute
// THEN rounds down to "0m"
assertEqual(formatDuration(30), "0m", "formatDuration(30s)");

// WHEN formatting exact minutes
// THEN returns minutes only
assertEqual(formatDuration(60), "1m", "formatDuration(60s)");
assertEqual(formatDuration(150), "2m", "formatDuration(150s)");

// WHEN formatting exact hours
// THEN returns hours only
assertEqual(formatDuration(3600), "1h", "formatDuration(1h)");
assertEqual(formatDuration(7200), "2h", "formatDuration(2h)");

// WHEN formatting hours and minutes
// THEN returns combined format
assertEqual(formatDuration(3660), "1h1m", "formatDuration(1h1m)");
assertEqual(formatDuration(5400), "1h30m", "formatDuration(1h30m)");
assertEqual(formatDuration(9120), "2h32m", "formatDuration(2h32m)");

// --- parseDurationToSec ---

// WHEN parsing Go duration strings
// THEN returns correct total seconds
assertEqual(parseDurationToSec("10s"), 10, "parseDurationToSec(10s)");
assertEqual(parseDurationToSec("1m0s"), 60, "parseDurationToSec(1m0s)");
assertEqual(parseDurationToSec("2m"), 120, "parseDurationToSec(2m)");
assertEqual(parseDurationToSec("1h0m0s"), 3600, "parseDurationToSec(1h0m0s)");
assertEqual(parseDurationToSec("1h30m0s"), 5400, "parseDurationToSec(1h30m0s)");
assertEqual(parseDurationToSec("5m0s"), 300, "parseDurationToSec(5m0s)");

// --- formatRangeLabel ---

// WHEN formatting time range labels
// THEN uses appropriate unit (m, h, d, mo, y)
assertEqual(formatRangeLabel(30), "30m", "formatRangeLabel(30)");
assertEqual(formatRangeLabel(60), "1h", "formatRangeLabel(60)");
assertEqual(formatRangeLabel(240), "4h", "formatRangeLabel(240)");
assertEqual(formatRangeLabel(1440), "1d", "formatRangeLabel(1440)");
assertEqual(formatRangeLabel(2880), "2d", "formatRangeLabel(2880)");
assertEqual(formatRangeLabel(43200), "1mo", "formatRangeLabel(43200)");
assertEqual(formatRangeLabel(525600), "1y", "formatRangeLabel(525600)");

// --- alignedStart ---

const vnOffsetMs = 7 * 60 * 60000;
const t134914 = new Date("2026-04-03T06:49:14Z").getTime(); // 13:49:14 +07:00
const t134000 = new Date("2026-04-03T06:40:00Z").getTime(); // 13:40:00 +07:00
const t130000 = new Date("2026-04-03T06:00:00Z").getTime(); // 13:00:00 +07:00

// WHEN snapping 13:49:14 with 10-minute buckets
// THEN aligns to 13:40:00
assertEqual(alignedStart(t134914, 600000, vnOffsetMs), t134000,
	"alignedStart 10m buckets");

// WHEN snapping 13:49:14 with 1-hour buckets
// THEN aligns to 13:00:00
assertEqual(alignedStart(t134914, 3600000, vnOffsetMs), t130000,
	"alignedStart 1h buckets");

// --- calcActiveDuration (tick-based) ---

// GIVEN a single ACTIVE entry at 10:00 with 10s poll
// WHEN calculating active duration over [10:00, 10:00:10)
// THEN active is 10s (one tick), total is 10s
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:10Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 10, "calcActive: single tick 10s active");
	assertEqual(result.totalSec, 10, "calcActive: single tick 10s total");
}

// GIVEN ACTIVE at 10:00 and IDLE at 10:00:10
// WHEN calculating over [10:00, 10:00:20)
// THEN active is 10s (first tick), total is 20s
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:10+07:00", State: "IDLE"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:20Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 10, "calcActive: active+idle 10s active");
	assertEqual(result.totalSec, 20, "calcActive: active+idle 20s total");
}

// GIVEN two ACTIVE entries 10s apart (raw consecutive)
// WHEN calculating over [10:00, 10:00:20)
// THEN active is 20s (two ticks), total is 20s
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:10+07:00", State: "ACTIVE"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:20Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 20, "calcActive: two raw ticks 20s active");
	assertEqual(result.totalSec, 20, "calcActive: two raw ticks 20s total");
}

// GIVEN a gap: ACTIVE at 10:00, then ACTIVE at 10:05 (5m gap)
// WHEN calculating over [10:00, 10:05:10)
// THEN active is 20s (two ticks), gap is idle, total is 5m10s
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:05:00+07:00", State: "ACTIVE"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:05:10Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 20, "calcActive: gap counted as idle, 20s active");
	assertEqual(result.totalSec, 310, "calcActive: gap total 5m10s");
}

// GIVEN a compact ACTIVE entry from 10:00 to 10:59:50
// WHEN calculating over [10:00, 11:00)
// THEN active is 1h (compact covers [10:00, 10:59:50+10s) = [10:00, 11:00))
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE", IsCompact: true, TimeCompactEnd: "2026-04-03T10:59:50+07:00"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 3600, "calcActive: compact 1h active");
	assertEqual(result.totalSec, 3600, "calcActive: compact 1h total");
}

// GIVEN compact ACTIVE 08:00-08:59:50 then raw IDLE at 09:00
// WHEN calculating over [08:00, 09:00:10)
// THEN active is 1h (compact), total is 1h10s
{
	const entries = [
		{Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE", IsCompact: true, TimeCompactEnd: "2026-04-03T08:59:50+07:00"},
		{Time: "2026-04-03T09:00:00+07:00", State: "IDLE"},
	];
	const rangeStart = new Date("2026-04-03T01:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T02:00:10Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 3600, "calcActive: compact+idle 1h active");
	assertEqual(result.totalSec, 3610, "calcActive: compact+idle 1h10s total");
}

// GIVEN data starts at 10:00 but range starts at 09:00
// WHEN calculating over [09:00, 10:00:10)
// THEN active is 10s, total is 1h10s (full range window)
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
	];
	const rangeStart = new Date("2026-04-03T02:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:10Z").getTime();
	const result = calcActiveDuration(entries, rangeStart, rangeEnd, pollMs);
	assertEqual(result.activeSec, 10, "calcActive: range wider than data, 10s active");
	assertEqual(result.totalSec, 3610, "calcActive: range wider than data, 1h10s total");
}

// --- active percentage ---

// GIVEN ACTIVE and IDLE entries each covering one tick
// WHEN calculating percentage over [10:00, 10:00:20)
// THEN 50%
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:10+07:00", State: "IDLE"},
	];
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:20Z").getTime();
	assertEqual(activePct(entries, rangeStart, rangeEnd), 50, "activePct: 50% one tick each");
}

// GIVEN only ACTIVE entries (no gaps)
// WHEN calculating percentage
// THEN 100%
{
	const entries = [
		{Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T08:00:10+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T08:00:20+07:00", State: "ACTIVE"},
	];
	const rangeStart = new Date("2026-04-03T01:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T01:00:30Z").getTime();
	assertEqual(activePct(entries, rangeStart, rangeEnd), 100, "activePct: 100% consecutive active");
}

// GIVEN only IDLE entries
// WHEN calculating percentage
// THEN 0%
{
	const entries = [
		{Time: "2026-04-03T08:00:00+07:00", State: "IDLE"},
	];
	const rangeStart = new Date("2026-04-03T01:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T01:00:10Z").getTime();
	assertEqual(activePct(entries, rangeStart, rangeEnd), 0, "activePct: 0% idle");
}

// WHEN no entries
// THEN 0%
assertEqual(activePct([], 0, Date.now()), 0, "activePct: 0% empty");

// GIVEN ACTIVE entry with a gap before rangeEnd
// WHEN calculating percentage
// THEN gap reduces active percentage
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
	];
	// rangeEnd is 20s after entry (10s active tick + 10s gap)
	const rangeStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:00:20Z").getTime();
	assertEqual(activePct(entries, rangeStart, rangeEnd), 50, "activePct: gap reduces pct");
}

// --- bucketizeEntries (tick-based) ---

// GIVEN 3 consecutive ACTIVE entries (10s each) in the same 1-minute bucket
// WHEN bucketing into 1m buckets
// THEN the bucket has 30s active
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:10+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:20+07:00", State: "ACTIVE"},
	];
	const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:01:00Z").getTime();
	const b = bucketizeEntries(entries, gridStart, 60000, rangeEnd, pollMs);
	assertEqual(b[0].activeSec, 30, "bucketize tick: 3 ticks = 30s active");
}

// GIVEN ACTIVE then IDLE entries in one bucket
// WHEN bucketing
// THEN the bucket has correct active and idle
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:10+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:20+07:00", State: "IDLE"},
		{Time: "2026-04-03T10:00:30+07:00", State: "IDLE"},
	];
	const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:01:00Z").getTime();
	const b = bucketizeEntries(entries, gridStart, 60000, rangeEnd, pollMs);
	assertEqual(b[0].activeSec, 20, "bucketize tick: 20s active");
	assertEqual(b[0].idleSec, 20, "bucketize tick: 20s idle");
}

// GIVEN a compact ACTIVE entry spanning two 30m buckets
// WHEN bucketing into 30m buckets
// THEN each bucket gets the correct share
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE", IsCompact: true, TimeCompactEnd: "2026-04-03T10:59:50+07:00"},
	];
	const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
	const b = bucketizeEntries(entries, gridStart, 30 * 60 * 1000, rangeEnd, pollMs);
	assertEqual(b[0].activeSec, 1800, "bucketize compact: first 30m all active");
	assertEqual(b[1].activeSec, 1800, "bucketize compact: second 30m all active");
}

// GIVEN entries with a gap (ACTIVE at :00, nothing until :50, ACTIVE at :50)
// WHEN bucketing into 1m buckets
// THEN only the ticks are counted, gap time is not distributed
{
	const entries = [
		{Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
		{Time: "2026-04-03T10:00:50+07:00", State: "ACTIVE"},
	];
	const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
	const rangeEnd = new Date("2026-04-03T03:01:00Z").getTime();
	const b = bucketizeEntries(entries, gridStart, 60000, rangeEnd, pollMs);
	assertEqual(b[0].activeSec, 20, "bucketize gap: only 2 ticks = 20s, not 60s");
}

// Summary
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) {
	process.exit(1);
}
