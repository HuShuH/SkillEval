const state = {
  selectedRunID: "",
  selectedReport: null,
  eventSource: null,
  runFilter: "all",
  selectedCaseID: "",
  selectedSide: "",
  caseFilter: "all",
  eventFilter: "all",
  currentEvents: [],
  liveEvents: [],
};

const els = {
  runsStatus: document.getElementById("runs-status"),
  runsList: document.getElementById("runs-list"),
  runFilter: document.getElementById("run-filter"),
  reportStatus: document.getElementById("report-status"),
  reportAlerts: document.getElementById("report-alerts"),
  reportSummary: document.getElementById("report-summary"),
  casesList: document.getElementById("cases-list"),
  caseFilter: document.getElementById("case-filter"),
  eventsStatus: document.getElementById("events-status"),
  eventsList: document.getElementById("events-list"),
  eventFilter: document.getElementById("event-filter"),
  liveStatus: document.getElementById("live-status"),
  liveEvents: document.getElementById("live-events"),
  refreshRuns: document.getElementById("refresh-runs"),
  connectStream: document.getElementById("connect-stream"),
};

els.refreshRuns.addEventListener("click", loadRuns);
els.connectStream.addEventListener("click", connectStream);
els.runFilter.addEventListener("change", () => {
  state.runFilter = els.runFilter.value;
  loadRuns();
});
els.caseFilter.addEventListener("change", () => {
  state.caseFilter = els.caseFilter.value;
  if (state.selectedReport) {
    renderCases(state.selectedReport);
  }
});
els.eventFilter.addEventListener("change", () => {
  state.eventFilter = els.eventFilter.value;
  renderEventList(els.eventsList, state.currentEvents);
  renderEventList(els.liveEvents, state.liveEvents);
});

loadRuns();

async function loadRuns() {
  els.runsStatus.textContent = "Loading runs...";
  els.runsList.innerHTML = "";
  try {
    const query = buildRunsQuery();
    const data = await getJSON(`/api/runs${query}`);
    const runs = data.runs || [];
    els.runsStatus.textContent = runs.length ? `${runs.length} run(s)` : "No runs found.";
    for (const run of runs) {
      const card = document.createElement("div");
      card.className = `card clickable ${runStatusClass(run)}`;
      card.innerHTML = `
        <strong>${escapeHTML(run.run_id)}</strong>
        <div class="meta">${escapeHTML(run.created_at || "unknown time")}</div>
        <div class="meta">mode: ${escapeHTML(run.mode || "unknown")} · cases: ${run.total_cases || 0}</div>
        <div class="meta">provider: ${escapeHTML(run.provider || "-")} · model: ${escapeHTML(run.model || "-")}</div>
        <div class="meta">failed: ${run.failed || 0} · errored: ${run.errored || 0} · timed out: ${run.timed_out_count || 0}</div>
        <div class="meta">html: ${run.has_html_report ? "yes" : "no"}</div>
      `;
      card.addEventListener("click", () => loadReport(run.run_id));
      els.runsList.appendChild(card);
    }
    if (data.skipped && data.skipped.length) {
      els.runsStatus.textContent += ` · skipped ${data.skipped.length} bad run(s)`;
    }
  } catch (error) {
    els.runsStatus.innerHTML = `<span class="error">${escapeHTML(error.message)}</span>`;
  }
}

async function loadReport(runID) {
  state.selectedRunID = runID;
  state.selectedReport = null;
  els.reportStatus.textContent = `Loading ${runID}...`;
  els.reportAlerts.innerHTML = "";
  els.reportSummary.innerHTML = "";
  els.casesList.innerHTML = "";
  els.eventsList.innerHTML = "";
  state.currentEvents = [];
  state.selectedCaseID = "";
  state.selectedSide = "";
  els.eventsStatus.textContent = "Select a case.";
  els.liveStatus.textContent = "Run selected. Connect stream if it is live.";

  try {
    const report = await getJSON(`/api/runs/${encodeURIComponent(runID)}`);
    state.selectedReport = report;
    els.reportStatus.textContent = `Report ${runID}`;
    renderSummary(report);
    renderCases(report);
  } catch (error) {
    els.reportStatus.innerHTML = `<span class="error">${escapeHTML(error.message)}</span>`;
  }
}

function renderSummary(report) {
  const summary = report.summary || {};
  const isPair = Object.prototype.hasOwnProperty.call(summary, "total_pairs");
  const fields = isPair
    ? [
        ["total pairs", summary.total_pairs],
        ["both passed", summary.both_passed],
        ["only A passed", summary.only_a_passed],
        ["only B passed", summary.only_b_passed],
        ["both failed", summary.both_failed],
        ["errored pairs", summary.errored_pairs],
        ["A avg iterations", summary.a?.average_iterations],
        ["B avg iterations", summary.b?.average_iterations],
        ["A tool calls", summary.a?.total_tool_calls],
        ["B tool calls", summary.b?.total_tool_calls],
      ]
    : [
        ["total cases", summary.total_cases],
        ["passed", summary.passed],
        ["failed", summary.failed],
        ["unchecked", summary.unchecked],
        ["errored", summary.errored],
        ["timed out", summary.timed_out_count],
        ["canceled", summary.canceled_count],
        ["avg iterations", summary.average_iterations],
        ["tool calls", summary.total_tool_calls],
      ];

  const alertFragments = [];
  if (!isPair && (summary.errored || summary.timed_out_count || summary.canceled_count)) {
    alertFragments.push(`Errored: ${summary.errored || 0}`);
    alertFragments.push(`Timed out: ${summary.timed_out_count || 0}`);
    alertFragments.push(`Canceled: ${summary.canceled_count || 0}`);
  }
  if (!isPair && summary.error_classes && Object.keys(summary.error_classes).length) {
    const topClasses = Object.entries(summary.error_classes)
      .map(([key, value]) => `${key}: ${value}`)
      .join(" · ");
    alertFragments.push(`Error classes: ${topClasses}`);
  }
  els.reportAlerts.innerHTML = alertFragments.length
    ? `<div class="alert error-alert">${alertFragments.map(escapeHTML).join("<br />")}</div>`
    : "";

  els.reportSummary.innerHTML = `
    <div class="summary-grid">
      ${fields.map(([key, value]) => `<div class="kv"><strong>${escapeHTML(key)}</strong>${escapeHTML(formatValue(value))}</div>`).join("")}
    </div>
    ${summary.stop_reasons ? `<h3>Stop Reasons</h3><pre>${escapeHTML(JSON.stringify(summary.stop_reasons, null, 2))}</pre>` : ""}
    ${summary.error_classes ? `<h3>Error Classes</h3><pre>${escapeHTML(JSON.stringify(summary.error_classes, null, 2))}</pre>` : ""}
  `;
}

function renderCases(report) {
  const results = report.results || [];
  els.casesList.innerHTML = "";
  const filtered = results.filter((result) => caseMatchesFilter(result, state.caseFilter));
  for (const result of filtered) {
    const isPair = result.a && result.b;
    const caseID = result.case_id || result.case_result?.case_id;
    const card = document.createElement("div");
    card.className = `card ${caseSeverityClass(result)}`;
    if (isPair) {
      card.innerHTML = `
        <strong>${escapeHTML(caseID)}</strong>
        <div class="meta">A: ${escapeHTML(result.a.case_result?.stop_reason || "unknown")} · B: ${escapeHTML(result.b.case_result?.stop_reason || "unknown")}</div>
        <div class="meta">A error: ${escapeHTML(result.a.case_result?.error_class || "-")} · B error: ${escapeHTML(result.b.case_result?.error_class || "-")}</div>
        <button data-side="a">Events A</button>
        <button data-side="b">Events B</button>
      `;
      card.querySelector('[data-side="a"]').addEventListener("click", () => loadEvents(caseID, "a"));
      card.querySelector('[data-side="b"]').addEventListener("click", () => loadEvents(caseID, "b"));
    } else {
      card.classList.add("clickable");
      card.innerHTML = `
        <strong>${escapeHTML(caseID)}</strong>
        <div class="meta">passed: ${Boolean(result.passed)} · stop: ${escapeHTML(result.stop_reason || "unknown")}</div>
        <div class="meta">error: ${escapeHTML(result.error_class || "-")} · failed iteration: ${escapeHTML(formatValue(result.failed_iteration || 0))}</div>
      `;
      card.addEventListener("click", () => loadEvents(caseID, ""));
    }
    els.casesList.appendChild(card);
  }
  if (!filtered.length) {
    els.casesList.innerHTML = `<div class="status">No cases match the current filter.</div>`;
  }
}

async function loadEvents(caseID, side) {
  if (!state.selectedRunID) return;
  state.selectedCaseID = caseID;
  state.selectedSide = side;
  els.eventsStatus.textContent = `Loading events for ${caseID}${side ? ` (${side})` : ""}...`;
  els.eventsList.innerHTML = "";
  const suffix = side ? `?side=${encodeURIComponent(side)}` : "";
  try {
    const data = await getJSON(`/api/runs/${encodeURIComponent(state.selectedRunID)}/cases/${encodeURIComponent(caseID)}/events${suffix}`);
    const events = data.events || [];
    state.currentEvents = events;
    els.eventsStatus.textContent = `${events.length} event(s)`;
    renderEventList(els.eventsList, events);
  } catch (error) {
    els.eventsStatus.innerHTML = `<span class="error">${escapeHTML(error.message)}</span>`;
  }
}

function connectStream() {
  if (!state.selectedRunID) {
    els.liveStatus.textContent = "Select a run first.";
    return;
  }
  if (state.eventSource) {
    state.eventSource.close();
  }

  state.liveEvents = [];
  els.liveEvents.innerHTML = "";
  els.liveStatus.textContent = "Connecting...";
  const source = new EventSource(`/api/runs/${encodeURIComponent(state.selectedRunID)}/stream`);
  state.eventSource = source;

  source.onopen = () => {
    els.liveStatus.textContent = "Connected.";
  };
  source.onerror = () => {
    els.liveStatus.textContent = "Not live or stream closed.";
    source.close();
  };
  source.addEventListener("message", (event) => {
    const parsed = JSON.parse(event.data);
    state.liveEvents.push({
      type: parsed.event_type,
      iteration: parsed.event?.iteration,
      timestamp: parsed.timestamp,
      message: parsed.event?.message,
      metadata: {
        run_id: parsed.run_id,
        case_id: parsed.case_id,
        side: parsed.side,
        ...(parsed.event?.metadata || {}),
      },
    });
    renderEventList(els.liveEvents, state.liveEvents);
  });
}

async function getJSON(url) {
  const response = await fetch(url);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || `request failed: ${response.status}`);
  }
  return data;
}

function renderEventList(container, events) {
  container.innerHTML = "";
  const filtered = events.filter((event) => eventMatchesFilter(event, state.eventFilter));
  for (const event of filtered) {
    appendEvent(container, event);
  }
  if (!filtered.length) {
    container.innerHTML = `<div class="status">No events match the current filter.</div>`;
  }
}

function appendEvent(container, event) {
  const div = document.createElement("div");
  const severity = eventSeverityClass(event);
  div.className = `event ${severity}`;
  const metadata = event.metadata || {};
  const details = [];
  if (metadata.error_class) details.push(`error_class=${metadata.error_class}`);
  if (metadata.status_code) details.push(`status=${metadata.status_code}`);
  if (metadata.attempt) details.push(`attempt=${metadata.attempt}`);
  if (metadata.next_attempt) details.push(`next_attempt=${metadata.next_attempt}`);
  if (metadata.backoff_ms) details.push(`backoff_ms=${metadata.backoff_ms}`);
  if (metadata.tool) details.push(`tool=${metadata.tool}`);
  if (metadata.operation) details.push(`operation=${metadata.operation}`);

  div.innerHTML = `
    <div class="event-title">
      <span class="badge ${severity}">${escapeHTML(event.type || "event")}</span>
      <span class="meta">iteration ${escapeHTML(formatValue(event.iteration ?? 0))}</span>
      <span class="meta">${escapeHTML(event.timestamp || "")}</span>
    </div>
    ${event.message ? `<div class="event-message">${escapeHTML(event.message)}</div>` : ""}
    ${details.length ? `<div class="event-inline">${escapeHTML(details.join(" · "))}</div>` : ""}
    ${Object.keys(metadata).length ? `<details><summary>metadata</summary><pre>${escapeHTML(JSON.stringify(metadata, null, 2))}</pre></details>` : ""}
  `;
  container.appendChild(div);
  container.scrollTop = container.scrollHeight;
}

function caseMatchesFilter(result, filter) {
  if (filter === "all") return true;
  if (result.a && result.b) {
    const a = result.a.case_result || {};
    const b = result.b.case_result || {};
    if (filter === "failed") {
      return Boolean(result.error || a.error || b.error || !a.passed || !b.passed);
    }
    if (filter === "timed") {
      return [a.stop_reason, b.stop_reason].some((value) => value === "timed_out" || value === "canceled");
    }
    return true;
  }
  if (filter === "failed") {
    return Boolean(result.error || !result.passed);
  }
  if (filter === "timed") {
    return result.stop_reason === "timed_out" || result.stop_reason === "canceled";
  }
  return true;
}

function caseSeverityClass(result) {
  if (result.a && result.b) {
    const reasons = [result.a.case_result?.stop_reason, result.b.case_result?.stop_reason];
    if (reasons.includes("timed_out") || reasons.includes("canceled")) return "state-warning";
    if (result.error || result.a.case_result?.error || result.b.case_result?.error) return "state-error";
    return "state-ok";
  }
  if (result.stop_reason === "timed_out" || result.stop_reason === "canceled") return "state-warning";
  if (result.error || result.error_class || result.passed === false) return "state-error";
  return "state-ok";
}

function eventMatchesFilter(event, filter) {
  if (filter === "all") return true;
  if (filter === "errors") {
    return isErrorEvent(event);
  }
  if (filter === "retries") {
    return event.type === "provider.request.retried";
  }
  return true;
}

function isErrorEvent(event) {
  return [
    "provider.request.failed",
    "tool.validation.failed",
    "run.timed_out",
    "run.canceled",
    "run.failed",
  ].includes(event.type);
}

function eventSeverityClass(event) {
  if (event.type === "provider.request.retried") return "event-retry";
  if (event.type === "run.timed_out" || event.type === "run.canceled") return "event-warning";
  if (isErrorEvent(event)) return "event-error";
  return "event-normal";
}

function formatValue(value) {
  if (value === undefined || value === null) return "0";
  if (typeof value === "number") return Number.isInteger(value) ? String(value) : value.toFixed(2);
  return String(value);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function buildRunsQuery() {
  const params = new URLSearchParams();
  switch (state.runFilter) {
    case "failed":
    case "errored":
    case "timed_out":
      params.set("status", state.runFilter);
      break;
    case "single":
    case "pair":
      params.set("mode", state.runFilter);
      break;
    default:
      break;
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

function runStatusClass(run) {
  if ((run.timed_out_count || 0) > 0 || (run.canceled_count || 0) > 0) {
    return "state-warning";
  }
  if ((run.errored || 0) > 0 || (run.failed || 0) > 0) {
    return "state-error";
  }
  return "state-ok";
}
