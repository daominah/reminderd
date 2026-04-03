// Run with: node web/chart_calc_test.js

const { formatDuration, parseDurationToSec, formatRangeLabel, alignedStart, calcActiveDuration, bucketizeEntries } = require("./chart_calc");

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

function activePct(entries, rangeEndMs) {
  const { activeSec, totalSec } = calcActiveDuration(entries, rangeEndMs);
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

// --- calcActiveDuration ---

// GIVEN a user active 10:00-10:30, then idle 10:30-11:00
// WHEN calculating active duration
// THEN active is 30m, total is 1h
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:30:00+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 1800, "calcActiveDuration: 30m active");
  assertEqual(result.totalSec, 3600, "calcActiveDuration: 1h total");
}

// GIVEN compacted data with two ACTIVE entries spanning 1h, then idle
// WHEN calculating active duration
// THEN the full active span is counted
{
  const entries = [
    {Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T09:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T09:00:10+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T02:30:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 3610, "calcActiveDuration compacted: 1h0m10s active");
}

// --- bucketizeEntries ---

// GIVEN 1h of continuous activity
// WHEN bucketing into two 30m buckets
// THEN each bucket gets 30m of active time
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
  ];
  const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  const b = bucketizeEntries(entries, gridStart, 30 * 60 * 1000, rangeEnd);
  assertEqual(b[0].activeSec, 1800, "bucketize: first 30m all active");
  assertEqual(b[1].activeSec, 1800, "bucketize: second 30m all active");
}

// GIVEN 20m active then 10m idle within one 30m bucket
// WHEN bucketing
// THEN the bucket has 20m active and 10m idle
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:20:00+07:00", State: "IDLE"},
  ];
  const gridStart = new Date("2026-04-03T03:00:00Z").getTime();
  const rangeEnd = new Date("2026-04-03T03:30:00Z").getTime();
  const b = bucketizeEntries(entries, gridStart, 30 * 60 * 1000, rangeEnd);
  assertEqual(b[0].activeSec, 1200, "bucketize: 20m active");
  assertEqual(b[0].idleSec, 600, "bucketize: 10m idle");
}

// --- active percentage ---

// GIVEN 30m active then 30m idle
// WHEN calculating percentage
// THEN 50%
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:30:00+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  assertEqual(activePct(entries, rangeEnd), 50, "activePct: 50%");
}

// GIVEN only active entries
// WHEN calculating percentage
// THEN 100%
{
  const entries = [
    {Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE"},
  ];
  const rangeEnd = new Date("2026-04-03T02:00:00Z").getTime();
  assertEqual(activePct(entries, rangeEnd), 100, "activePct: 100%");
}

// GIVEN only idle entries
// WHEN calculating percentage
// THEN 0%
{
  const entries = [
    {Time: "2026-04-03T08:00:00+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T02:00:00Z").getTime();
  assertEqual(activePct(entries, rangeEnd), 0, "activePct: 0%");
}

// GIVEN multiple active-idle sessions (1h active, 1h idle, 30m active, 30m idle)
// WHEN calculating percentage
// THEN 50%
{
  const entries = [
    {Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T09:00:00+07:00", State: "IDLE"},
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:30:00+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  assertEqual(activePct(entries, rangeEnd), 50, "activePct: 50% multiple sessions");
}

// GIVEN compacted data with a 2h active span then 1h idle
// WHEN calculating percentage
// THEN 67%
{
  const entries = [
    {Time: "2026-04-03T08:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T10:00:10+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  assertEqual(activePct(entries, rangeEnd), 67, "activePct: 67% compacted");
}

// WHEN no entries
// THEN 0%
assertEqual(activePct([], Date.now()), 0, "activePct: 0% empty");

// --- edge cases ---

// GIVEN the last ACTIVE entry is 5 seconds before rangeEnd
// WHEN calculating active duration
// THEN active is 5s, total is 5s
{
  const entries = [
    {Time: "2026-04-03T14:59:55+07:00", State: "ACTIVE"},
  ];
  const rangeEnd = new Date("2026-04-03T08:00:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 5, "edge: 5s before rangeEnd");
  assertEqual(result.totalSec, 5, "edge: total 5s");
}

// GIVEN the last entry timestamp equals rangeEnd exactly
// WHEN calculating active duration
// THEN the last entry contributes zero duration
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T11:00:00+07:00", State: "ACTIVE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 3600, "edge: last entry at rangeEnd");
}

// GIVEN a single entry at exactly rangeEnd
// WHEN calculating active duration
// THEN zero duration
{
  const entries = [
    {Time: "2026-04-03T11:00:00+07:00", State: "ACTIVE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 0, "edge: single entry at rangeEnd");
  assertEqual(result.totalSec, 0, "edge: zero total at rangeEnd");
}

// GIVEN an entry timestamp after rangeEnd (clock skew)
// WHEN calculating active duration
// THEN the span is capped at rangeEnd
{
  const entries = [
    {Time: "2026-04-03T10:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T11:00:05+07:00", State: "IDLE"},
  ];
  const rangeEnd = new Date("2026-04-03T04:00:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 3600, "edge: clock skew capped at rangeEnd");
}

// GIVEN the user has been active since 14:00, now is 14:48
// WHEN querying "Last 1h" (range starts 13:48, data starts at 14:00)
// THEN active is 48m, percentage is 100% (no idle data exists)
{
  const entries = [
    {Time: "2026-04-03T14:00:00+07:00", State: "ACTIVE"},
    {Time: "2026-04-03T14:48:00+07:00", State: "ACTIVE"},
  ];
  const rangeEnd = new Date("2026-04-03T07:48:00Z").getTime();
  const result = calcActiveDuration(entries, rangeEnd);
  assertEqual(result.activeSec, 2880, "real scenario: 48m active");
  assertEqual(activePct(entries, rangeEnd), 100, "real scenario: 100%");
}

// Summary
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
