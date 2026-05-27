(function () {
  var mediaTypes = ["audio", "video", "location", "metadata"];

  function basePath() {
    return window.location.pathname.replace(/\/$/, "");
  }

  function dataPath() {
    return basePath() + "/data";
  }

  function element(tagName, className) {
    var node = document.createElement(tagName);
    if (className) {
      node.className = className;
    }
    return node;
  }

  function clearElement(node) {
    while (node.firstChild) {
      node.removeChild(node.firstChild);
    }
  }

  function setValue(selector, value, muted) {
    var node = document.querySelector(selector);
    if (!node) {
      return;
    }
    node.textContent = value;
    node.classList.toggle("muted", Boolean(muted));
  }

  function appendLabeledItem(grid, label, value, muted) {
    var item = element("div", "item");
    var labelNode = element("div", "label");
    var valueNode = element("div", muted ? "value muted" : "value");
    labelNode.textContent = label;
    valueNode.textContent = value;
    item.appendChild(labelNode);
    item.appendChild(valueNode);
    grid.appendChild(item);
  }

  function appendCell(row, value, muted) {
    var cell = element("td", muted ? "muted" : "");
    cell.textContent = value;
    row.appendChild(cell);
  }

  function isNumber(value) {
    return typeof value === "number" && Number.isFinite(value);
  }

  function parseTime(value) {
    if (!value) {
      return null;
    }
    var parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return null;
    }
    return parsed;
  }

  function plural(value, unit) {
    return value === 1 ? "1 " + unit : value + " " + unit + "s";
  }

  function humanDuration(milliseconds) {
    var minutes = Math.floor(milliseconds / 60000);
    if (minutes < 1) {
      return "less than a minute";
    }
    if (minutes < 60) {
      return plural(minutes, "minute");
    }

    var hours = Math.floor(milliseconds / 3600000);
    if (hours < 24) {
      return plural(hours, "hour");
    }

    var days = Math.floor(hours / 24);
    if (days < 7) {
      return plural(days, "day");
    }

    var weeks = Math.floor(days / 7);
    if (weeks < 5) {
      return plural(weeks, "week");
    }

    var months = Math.floor(days / 30);
    if (months < 12) {
      return plural(months, "month");
    }
    return plural(Math.floor(days / 365), "year");
  }

  function relativeTime(value) {
    var parsed = parseTime(value);
    if (!parsed) {
      return "Unknown";
    }

    var difference = parsed.getTime() - Date.now();
    if (difference > 0) {
      return "in " + humanDuration(difference);
    }

    var elapsed = Math.abs(difference);
    if (elapsed < 60000) {
      return "just now";
    }
    return humanDuration(elapsed) + " ago";
  }

  function absoluteTime(value) {
    var parsed = parseTime(value);
    if (!parsed) {
      return "Unknown";
    }
    return new Intl.DateTimeFormat(undefined, {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    }).format(parsed);
  }

  function formatBytes(value) {
    if (!isNumber(value)) {
      return "0 B";
    }
    if (value < 1024) {
      return value + " B";
    }

    var divisor = 1024;
    var unitName = "KiB";
    while (value / divisor >= 1024) {
      divisor *= 1024;
      if (unitName === "KiB") {
        unitName = "MiB";
      } else if (unitName === "MiB") {
        unitName = "GiB";
      } else {
        break;
      }
    }
    return (value / divisor).toFixed(1) + " " + unitName;
  }

  function updateIncident(incident) {
    if (!incident) {
      return;
    }
    setValue("[data-incident-status]", incident.status || "Unknown", !incident.status);
    setValue("[data-incident-client-label]", incident.client_label || "Not provided", !incident.client_label);
    setValue("[data-incident-created]", absoluteTime(incident.created_at), !parseTime(incident.created_at));
    setValue("[data-incident-updated]", relativeTime(incident.updated_at), !parseTime(incident.updated_at));
  }

  function updateLatestCheckin(checkin) {
    var container = document.querySelector("[data-latest-checkin]");
    if (!container) {
      return;
    }
    clearElement(container);

    if (!checkin) {
      var empty = element("p", "muted");
      empty.textContent = "No checkins recorded.";
      container.appendChild(empty);
      return;
    }

    var grid = element("div", "grid");
    appendLabeledItem(grid, "Time", relativeTime(checkin.created_at), !parseTime(checkin.created_at));

    if (isNumber(checkin.device_battery_percent)) {
      appendLabeledItem(grid, "Battery", checkin.device_battery_percent + "%", false);
    } else {
      appendLabeledItem(grid, "Battery", "Unknown", true);
    }

    if (checkin.device_network) {
      appendLabeledItem(grid, "Network", checkin.device_network, false);
    } else {
      appendLabeledItem(grid, "Network", "Unknown", true);
    }

    if (isNumber(checkin.latitude) && isNumber(checkin.longitude)) {
      appendLabeledItem(grid, "Location", checkin.latitude + ", " + checkin.longitude, false);
    } else {
      appendLabeledItem(grid, "Location", "Unknown", true);
    }
    container.appendChild(grid);
  }

  function streamName(stream) {
    if (stream.label) {
      return stream.label;
    }
    if (stream.media_type) {
      return stream.media_type + " recording";
    }
    return "recording";
  }

  function streamSummary(stream) {
    var chunks = isNumber(stream.chunk_count) ? stream.chunk_count : 0;
    var completed = stream.completed_at ? relativeTime(stream.completed_at) : "recently";
    return plural(chunks, "chunk") + " \u00b7 " + formatBytes(stream.total_bytes) + " \u00b7 completed " + completed;
  }

  function updateCompletedRecordings(streams) {
    var container = document.querySelector("[data-completed-recordings]");
    if (!container) {
      return;
    }
    clearElement(container);

    if (!Array.isArray(streams) || streams.length === 0) {
      var empty = element("p", "muted");
      empty.textContent = "No completed recordings yet.";
      container.appendChild(empty);
      return;
    }

    var list = element("div", "stream-list");
    streams.forEach(function (stream) {
      var item = element("div", "stream-item");
      var copy = element("div", "");
      var title = element("div", "value");
      var meta = element("div", "muted");
      var link = element("a", "button");

      title.textContent = streamName(stream);
      meta.textContent = streamSummary(stream);
      link.textContent = "Download encrypted bundle";
      link.setAttribute("href", "#");
      link.setAttribute("data-stream-download", stream.id || "");

      copy.appendChild(title);
      copy.appendChild(meta);
      item.appendChild(copy);
      item.appendChild(link);
      list.appendChild(item);
    });
    container.appendChild(list);

    var paragraph = element("p", "");
    var incidentLink = element("a", "button secondary");
    incidentLink.textContent = "Download all completed bundles";
    incidentLink.setAttribute("href", "#");
    incidentLink.setAttribute("data-incident-download", "");
    paragraph.appendChild(incidentLink);
    container.appendChild(paragraph);
  }

  function mediaRows(media) {
    if (Array.isArray(media) && media.length > 0) {
      return media;
    }
    return mediaTypes.map(function (mediaType) {
      return { media_type: mediaType, chunk_count: 0 };
    });
  }

  function updateMediaRows(media) {
    var tbody = document.querySelector("[data-media-rows]");
    if (!tbody) {
      return;
    }
    clearElement(tbody);

    mediaRows(media).forEach(function (item) {
      var row = element("tr", "");
      var latest = item.latest_chunk;
      appendCell(row, item.media_type || "unknown", false);
      appendCell(row, isNumber(item.chunk_count) ? String(item.chunk_count) : "0", false);
      if (latest) {
        appendCell(row, "#" + latest.chunk_index + " \u00b7 " + formatBytes(latest.byte_size), false);
        appendCell(row, relativeTime(latest.created_at), !parseTime(latest.created_at));
      } else {
        appendCell(row, "None", true);
        appendCell(row, "None", true);
      }
      tbody.appendChild(row);
    });
  }

  function updateIncidentView(data) {
    window.__lastIncidentViewData = data;
    updateIncident(data.incident);
    updateLatestCheckin(data.latest_checkin);
    updateCompletedRecordings(data.completed_streams);
    updateMediaRows(data.media);
    updateDownloadLinks();
  }

  function updateDownloadLinks() {
    var base = basePath();
    document.querySelectorAll("[data-stream-download]").forEach(function (link) {
      var streamID = link.getAttribute("data-stream-download");
      link.setAttribute("href", base + "/streams/" + encodeURIComponent(streamID) + "/download");
    });
    document.querySelectorAll("[data-incident-download]").forEach(function (link) {
      link.setAttribute("href", base + "/incident/download");
    });
  }

  var latestPollRequestID = 0;

  function poll() {
    latestPollRequestID += 1;
    var requestID = latestPollRequestID;

    fetch(dataPath(), { cache: "no-store" })
      .then(function (response) {
        return response.ok ? response.json() : null;
      })
      .then(function (data) {
        if (data && requestID === latestPollRequestID) {
          updateIncidentView(data);
        }
      })
      .catch(function () {});
  }

  updateDownloadLinks();
  setInterval(poll, 10000);
})();
