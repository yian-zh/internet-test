/**
 * Design by fromScratch Studio - 2022, 2023 (fromscratch.io)
 * Implementation in HTML/CSS/JS by Timendus - 2024 (https://github.com/Timendus)
 *
 * See https://github.com/librespeed/speedtest/issues/585
 */

// States the UI can be in
const INITIALIZING = 0;
const READY = 1;
const RUNNING = 2;
const FINISHED = 3;

// Keep some global state here
const testState = {
  state: INITIALIZING,
  speedtest: null,
  servers: [],
  selectedServerDirty: false,
  testData: null,
  testDataDirty: false,
  telemetryEnabled: false,
};

// Bootstrap the application when the DOM is ready
window.addEventListener("DOMContentLoaded", async () => {
  createSpeedtest();
  hookUpButtons();
  startRenderingLoop();
  applyServerListJSON();
});

/**
 * Create a new Speedtest and hook it into the global state
 */
function createSpeedtest() {
  testState.speedtest = new Speedtest();
  
  testState.speedtest.onupdate = (data) => {
    testState.testData = data;
    testState.testDataDirty = true;
  };

  testState.speedtest.onend = (aborted) => {
    testState.state = aborted ? READY : FINISHED;

    // AUTOMATION PIPELINE: Automatically save metrics to cloud DB upon completion
    if (!aborted && testState.testData) {
      const payload = {
        ping_ms: parseFloat(testState.testData.pingStatus) || 0,
        jitter_ms: parseFloat(testState.testData.jitterStatus) || 0,
        download_mbps: parseFloat(testState.testData.dlStatus) || 0,
        upload_mbps: parseFloat(testState.testData.ulStatus) || 0,
        testing_platform: "LibreSpeed-DO (Singapore)"
      };

      console.log("Test sequence completed. Packaging telemetry for database write...", payload);

      // Transmit payload to the Go backend /save-results endpoint
      fetch('/save-results', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json' 
        },
        body: JSON.stringify(payload)
      })
      .then(response => {
        if (response.ok) {
          console.log("Telemetry matrix successfully logged to cloud PostgreSQL cluster!");
        } else {
          console.error("Database gateway rejected payload:", response.statusText);
        }
      })
      .catch(error => console.error("Network communication failure during transmission:", error));
    }
  };
}

/**
 * Make all the buttons respond to the right clicks
 */
function hookUpButtons() {
  document
    .querySelector("#start-button")
    .addEventListener("click", startButtonClickHandler);
  document
    .querySelector("#choose-privacy")
    .addEventListener("click", () =>
      document.querySelector("#privacy").showModal()
    );
  document
    .querySelector("#share-results")
    .addEventListener("click", () =>
      document.querySelector("#share").showModal()
    );
  document
    .querySelector("#copy-link")
    .addEventListener("click", copyLinkButtonClickHandler);
  document
    .querySelectorAll(".close-dialog, #close-privacy")
    .forEach((element) => {
      element.addEventListener("click", () =>
        document.querySelectorAll("dialog").forEach((modal) => modal.close())
      );
    });
}

/**
 * Event listener for clicks on the main start button
 */
function startButtonClickHandler() {
  switch (testState.state) {
    case READY:
    case FINISHED:
      testState.speedtest.start();
      testState.state = RUNNING;
      return;
    case RUNNING:
      testState.speedtest.abort();
      // testState.state is updated by `onend` handler of speedtest
      return;
    default:
      return;
  }
}

/**
 * Event listener for clicks on the "Copy link" button in the modal
 */
async function copyLinkButtonClickHandler() {
  const link = document.querySelector("img#results").src;
  await navigator.clipboard.writeText(link);
  const button = document.querySelector("#copy-link");
  button.classList.add("active");
  button.textContent = "Copied!";
  setTimeout(() => {
    button.classList.remove("active");
    button.textContent = "Copy link";
  }, 3000);
}

/**
 * Load server list from the configured source and populate the dropdown
 */
async function applyServerListJSON() {
  try {
    // Inlined target configurations. Absolute URL path bypasses worker ping issues.
    const servers = [
      {
        name: "Our Cloud Server (Singapore)",
        server: "https://king-prawn-app-wztk3.ondigitalocean.app/",
        dlURL: "download",
        ulURL: "upload",
        pingURL: "ping",
        getIpURL: "ping"
      }
    ];

    testState.servers = servers;

    // If there's only one server, just show it. No reachability checks needed.
    if (servers.length === 1) {
      populateDropdown(servers);
      return;
    }

    // For multiple servers: first run the built-in selection (which pings servers
    // and annotates them with pingT). Only then populate the dropdown so that
    // dead servers don't appear.
    testState.speedtest.addTestPoints(servers);
    testState.speedtest.selectServer((bestServer) => {
      const aliveServers = testState.servers.filter((s) => {
        if (s.pingT !== -1) return true;
        return typeof s.server === "string" && s.server.startsWith("//");
      });

      if (aliveServers.length > 0) {
        testState.servers = aliveServers;
      }
      populateDropdown(testState.servers);

      if (bestServer) {
        selectServer(bestServer);
      } else {
        alert(
          "Can't reach any of the speedtest servers! But you're on this page. Something weird is going on with your network."
        );
      }
    });
  } catch (error) {
    console.error("Failed to load server list:", error);
  }
}

/**
 * Add all the servers to the server selection dropdown and make it actually work.
 * @param {Array} servers - an array of server objects
 */
function populateDropdown(servers) {
  const serverSelector = document.querySelector("div.server-selector");
  const serverList = serverSelector.querySelector("ul.servers");

  serverSelector.classList.remove("single-server");
  serverSelector.classList.remove("active");
  serverList.classList.remove("active");
  serverList.innerHTML = "";

  if (servers.length === 1) {
    serverSelector.classList.add("single-server");
    selectServer(servers[0]);
    return;
  }
  serverSelector.classList.add("active");

  if (serverSelector.dataset.hooked !== "1") {
    serverSelector.dataset.hooked = "1";

    serverSelector.addEventListener("click", () => {
      serverList.classList.toggle("active");
    });
    document.addEventListener("click", (e) => {
      if (e.target.closest("div.server-selector") !== serverSelector)
        serverList.classList.remove("active");
    });
  }

  const parseServerName = (name) => {
    const parts = (name || "").split(",").map((s) => s.trim());
    let country, city;
    if (parts.length >= 3) {
      country = parts[1];
      city = parts[0];
    } else if (parts.length === 2) {
      country = parts[1];
      city = parts[0];
    } else {
      country = parts[0];
      city = "";
    }
    country = country.replace(/\s*\([^)]*\)\s*/g, "").trim();
    return { country, city };
  };
  const sorted = [...servers].sort((a, b) => {
    const pa = parseServerName(a.name);
    const pb = parseServerName(b.name);
    return pa.country.localeCompare(pb.country) || pa.city.localeCompare(pb.city);
  });

  sorted.forEach((server) => {
    const item = document.createElement("li");
    const link = document.createElement("a");
    link.href = "#";
    link.innerHTML = `${server.name}${
      server.sponsorName ? ` <span>(${server.sponsorName})</span>` : ""
    }`;
    link.addEventListener("click", () => selectServer(server));
    item.appendChild(link);
    serverList.appendChild(item);
  });
}

/**
 * Set the given server as the selected server for the speedtest
 * @param {Object} server - a server object
 */
function selectServer(server) {
  testState.speedtest.setSelectedServer(server);
  testState.selectedServerDirty = true;
  testState.state = READY;
}

/**
 * Start the requestAnimationFrame UI rendering loop
 */
function startRenderingLoop() {
  const serverSelector = document.querySelector("div.server-selector");
  const selectedServer = serverSelector.querySelector("#selected-server");
  const sponsor = serverSelector.querySelector("#sponsor");
  const startButton = document.querySelector("#start-button");
  const privacyWarning = document.querySelector("#privacy-warning");

  const gauges = document.querySelectorAll("#download-gauge, #upload-gauge");
  const downloadProgress = document.querySelector("#download-gauge .progress");
  const uploadProgress = document.querySelector("#upload-gauge .progress");
  const downloadGauge = document.querySelector("#download-gauge .speed");
  const uploadGauge = document.querySelector("#upload-gauge .speed");
  const downloadText = document.querySelector("#download-gauge span");
  const uploadText = document.querySelector("#upload-gauge span");

  const pingAndJitter = document.querySelectorAll(".ping, .jitter");
  const ping = document.querySelector("#ping");
  const jitter = document.querySelector("#jitter");
  const shareResults = document.querySelector("#share-results");
  const copyLink = document.querySelector("#copy-link");
  const resultsImage = document.querySelector("#results");

  const buttonTexts = {
    [INITIALIZING]: "Loading...",
    [READY]: "Let's start",
    [RUNNING]: "Abort",
    [FINISHED]: "Restart",
  };

  copyLink.classList.toggle("hidden", !navigator.clipboard);

  function renderUI() {
    startButton.textContent = buttonTexts[testState.state];
    startButton.classList.toggle("disabled", testState.state === INITIALIZING);
    startButton.classList.toggle("active", testState.state === RUNNING);

    serverSelector.classList.toggle("disabled", testState.state === RUNNING);

    if (testState.selectedServerDirty) {
      const server = testState.speedtest.getSelectedServer();
      selectedServer.textContent = server.name;
      if (server.sponsorName) {
        if (server.sponsorURL) {
          sponsor.innerHTML = `Sponsor: <a href="${server.sponsorURL}">${server.sponsorName}</a>`;
        } else {
          sponsor.textContent = `Sponsor: ${server.sponsorName}`;
        }
      } else {
        sponsor.innerHTML = "&nbsp;";
      }
      testState.selectedServerDirty = false;
    }

    gauges.forEach((e) =>
      e.classList.toggle(
        "enabled",
        testState.state === RUNNING || testState.state === FINISHED
      )
    );

    pingAndJitter.forEach((e) =>
      e.classList.toggle(
        "hidden",
        !(
          testState.testData &&
          testState.testData.pingStatus &&
          testState.testData.jitterStatus
        )
      )
    );

    shareResults.classList.toggle(
      "hidden",
      !(
        testState.state === FINISHED &&
        testState.telemetryEnabled &&
        testState.testData.testId
      )
    );

    if (testState.testDataDirty) {
      downloadProgress.style = `--progress-rotation: ${
        testState.testData.dlProgress * 180
      }deg`;
      uploadProgress.style = `--progress-rotation: ${
        testState.testData.ulProgress * 180
      }deg`;
      downloadGauge.style = `--speed-rotation: ${mbpsToRotation(
        testState.testData.dlStatus,
        testState.testData.testState === 1
      )}deg`;
      uploadGauge.style = `--speed-rotation: ${mbpsToRotation(
        testState.testData.ulStatus,
        testState.testData.testState === 3
      )}deg`;

      downloadText.textContent = numberToText(testState.testData.dlStatus);
      uploadText.textContent = numberToText(testState.testData.ulStatus);
      ping.textContent = numberToText(testState.testData.pingStatus);
      jitter.textContent = numberToText(testState.testData.jitterStatus);

      if (testState.testData.clientIp) {
        privacyWarning.innerHTML = '';
        const connectedThrough = document.createElement('span');
        connectedThrough.textContent = 'You are connected through:';
        const ipAddress = document.createTextNode(testState.testData.clientIp);

        privacyWarning.appendChild(connectedThrough);
        privacyWarning.appendChild(document.createElement('br'));
        privacyWarning.appendChild(ipAddress);
        privacyWarning.classList.remove("hidden");
      }

      if (testState.testData.testId) {
        resultsImage.src =
          window.location.href.substring(
            0,
            window.location.href.lastIndexOf("/")
          ) +
          "/results/?id=" +
          testState.testData.testId;
      }

      testState.testDataDirty = false;
    }

    requestAnimationFrame(renderUI);
  }

  renderUI();
}

/**
 * Convert a speed in Mbits per second to a rotation for the gauge
 */
function mbpsToRotation(speed, oscillate) {
  speed = Number(speed);
  if (speed <= 0) return 0;

  const minSpeed = 0;
  const maxSpeed = 10000;
  const minRotation = 0;
  const maxRotation = 180;

  const logMinSpeed = Math.log10(minSpeed + 1);
  const logMaxSpeed = Math.log10(maxSpeed + 1);
  const logSpeed = Math.log10(speed + 1);

  const power = (logSpeed - logMinSpeed) / (logMaxSpeed - logMinSpeed);
  const oscillation = oscillate ? 1 + 0.01 * Math.sin(Date.now() / 100) : 1;
  const rotation = power * oscillation * maxRotation;

  return Math.max(Math.min(rotation, maxRotation), minRotation);
}

/**
 * Convert a number to a user friendly version
 */
function numberToText(value) {
  if (!value) return "00";
  value = Number(value);
  if (value < 10) return value.toFixed(2);
  if (value < 100) return value.toFixed(1);
  return value.toFixed(0);
}
