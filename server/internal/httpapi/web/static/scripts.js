(function () {
  function dataPath() {
    return window.location.pathname.replace(/\/$/, "") + "/data";
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

  setInterval(poll, 10000);
})();
