(function () {
  function dataPath() {
    return window.location.pathname.replace(/\/$/, "") + "/data";
  }

  function updateDownloadLinks() {
    var base = window.location.pathname.replace(/\/$/, "");
    document.querySelectorAll("[data-stream-download]").forEach(function (link) {
      var streamID = link.getAttribute("data-stream-download");
      link.setAttribute("href", base + "/streams/" + encodeURIComponent(streamID) + "/download");
    });
    document.querySelectorAll("[data-incident-download]").forEach(function (link) {
      link.setAttribute("href", base + "/incident/download");
    });
  }

  function poll() {
    fetch(dataPath(), { cache: "no-store" })
      .then(function (response) {
        return response.ok ? response.json() : null;
      })
      .then(function (data) {
        if (data) {
          window.__lastEmergencyData = data;
        }
      })
      .catch(function () {});
  }

  updateDownloadLinks();
  setInterval(poll, 10000);
})();
