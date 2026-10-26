// EARTHROPY System — reads the central list of apps and shows what runs where.
(() => {
  const REGISTRY_URL = "/earthropy/apps.json";
  const $ = (sel) => document.querySelector(sel);

  const el = (tag, attrs = {}, children = []) => {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(attrs)) {
      if (key === "style") node.style.cssText = value;
      else if (key === "text") node.textContent = value;
      else if (value !== undefined && value !== null && value !== false) node.setAttribute(key, value);
    }
    for (const child of [].concat(children)) if (child) node.append(child);
    return node;
  };

  const appIcon = (app) =>
    el("span", { class: "app-icon", "aria-hidden": "true", style: `--from:${app.from};--to:${app.to}`, text: app.letter });

  /* ---------- theme ---------- */
  const themeLabel = $("#theme-label");
  const syncThemeLabel = () => {
    themeLabel.textContent = document.documentElement.dataset.theme === "dark" ? "Light mode" : "Dark mode";
  };
  $("#theme-toggle").addEventListener("click", () => {
    const dark = document.documentElement.dataset.theme !== "dark";
    if (dark) document.documentElement.dataset.theme = "dark";
    else delete document.documentElement.dataset.theme;
    try { localStorage.setItem("earthropy-theme", dark ? "dark" : "light"); } catch (e) {}
    syncThemeLabel();
  });
  syncThemeLabel();

  /* ---------- nav ---------- */
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.addEventListener("click", () => {
      document.querySelectorAll(".nav-item").forEach((other) => other.classList.toggle("active", other === item));
    });
  });

  /* ---------- server ---------- */
  $("#server-name").textContent = location.hostname || "server";
  $("#stat-address").textContent = location.host || "–";

  /* ---------- status check (same origin, through the doorman) ---------- */
  const checkOnline = async (url) => {
    try {
      const res = await fetch(url, { method: "GET", cache: "no-store", redirect: "manual", credentials: "same-origin" });
      return res.type === "opaqueredirect" || res.status < 500;
    } catch (e) {
      return false;
    }
  };

  /* ---------- app switcher ---------- */
  const brand = $("#brand");
  const switcher = $("#switcher");
  const backdrop = $("#switcher-backdrop");
  const openSwitcher = (open) => {
    switcher.hidden = !open;
    backdrop.hidden = !open;
    brand.setAttribute("aria-expanded", open ? "true" : "false");
    if (open) switcher.querySelector("a")?.focus();
  };
  brand.addEventListener("click", () => openSwitcher(switcher.hidden));
  backdrop.addEventListener("click", () => openSwitcher(false));
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") openSwitcher(false);
  });

  const renderSwitcher = (apps) => {
    const grid = $("#switcher-grid");
    grid.replaceChildren(
      ...apps.map((app) => {
        const installed = Boolean(app.url);
        const item = el(
          installed ? "a" : "span",
          {
            class: "switch-app",
            href: installed ? app.url : null,
            "data-installed": String(installed),
            "aria-current": app.id === "system" ? "page" : null,
            title: installed ? app.description : "Not installed yet",
          },
          [appIcon(app), el("span", { text: app.name })],
        );
        return item;
      }),
    );
  };

  /* ---------- apps grid ---------- */
  const renderApps = (apps, status) => {
    const grid = $("#apps-grid");
    grid.replaceChildren(
      ...apps
        .filter((app) => app.id !== "system")
        .map((app) => {
          const installed = Boolean(app.url);
          const state = !installed ? "idle" : status.get(app.id) === undefined ? "idle" : status.get(app.id) ? "ok" : "off";
          const label = !installed ? "Not installed yet" : state === "ok" ? "Online" : state === "off" ? "Offline" : "Checking…";
          return el("div", { class: "card app-card", "data-installed": String(installed) }, [
            el("div", { class: "app-card-head" }, [
              appIcon(app),
              el("div", {}, [el("div", { class: "app-card-name", text: app.name }), el("div", { class: "app-card-desc", text: app.description })]),
            ]),
            el("div", { class: "app-card-foot" }, [
              el("span", { class: "chip" }, [el("span", { class: `status-dot ${state === "idle" ? "" : state}` }), label]),
              installed
                ? el("a", { class: "pill pill--small", href: app.url, text: "Open" })
                : el("span", { class: "app-card-path", text: "coming soon" }),
            ]),
          ]);
        }),
    );
  };

  /* ---------- map ---------- */
  const renderMap = (apps) => {
    const routed = apps.filter((app) => app.url);
    const rowH = 64;
    const height = Math.max(routed.length * rowH + 40, 220);
    const NS = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", `0 0 920 ${height}`);
    svg.setAttribute("role", "img");
    svg.setAttribute("aria-label", "Map of the EARTHROPY server: the browser talks to the doorman on one port, which sends each path to its app");
    const add = (tag, attrs, text) => {
      const node = document.createElementNS(NS, tag);
      for (const [key, value] of Object.entries(attrs)) node.setAttribute(key, value);
      if (text) node.textContent = text;
      svg.append(node);
      return node;
    };

    const midY = height / 2;
    // browser
    add("rect", { class: "box", x: 0, y: midY - 34, width: 170, height: 68, rx: 16 });
    add("text", { x: 20, y: midY - 6, "font-size": 14, "font-weight": 700 }, "Your browser");
    add("text", { x: 20, y: midY + 16, "font-size": 12, class: "mono" }, location.host || "server:9090");
    // doorman
    add("path", { class: "wire", d: `M170 ${midY} L250 ${midY}` });
    add("rect", { class: "box", x: 250, y: midY - 46, width: 190, height: 92, rx: 16 });
    add("text", { x: 270, y: midY - 14, "font-size": 14, "font-weight": 700 }, "Doorman (Caddy)");
    add("text", { x: 270, y: midY + 8, "font-size": 12, class: "muted" }, "one port for everything");
    add("text", { x: 270, y: midY + 28, "font-size": 12, class: "mono" }, `:${location.port || "9090"}`);

    routed.forEach((app, index) => {
      const y = 20 + index * rowH + rowH / 2;
      add("path", { class: "wire", d: `M440 ${midY} C 500 ${midY}, 500 ${y}, 560 ${y}` });
      add("text", { x: 470, y: y - 8, "font-size": 11, class: "mono" }, app.url);
      add("rect", { class: "box", x: 560, y: y - 24, width: 200, height: 48, rx: 14 });
      const grad = add("linearGradient", { id: `g-${app.id}`, x1: 0, x2: 1, y1: 0, y2: 0 });
      grad.innerHTML = `<stop offset="0" stop-color="${app.from}"/><stop offset="1" stop-color="${app.to}"/>`;
      add("rect", { x: 570, y: y - 16, width: 32, height: 32, rx: 10, fill: `url(#g-${app.id})` });
      add("text", { x: 586, y: y + 5, "font-size": 13, "text-anchor": "middle", fill: "#fff", style: "fill:#fff;font-family:Orbitron,sans-serif" }, app.letter);
      add("text", { x: 614, y: y + 5, "font-size": 14, "font-weight": 600 }, app.name);
      // apps that use the files folder on the server
      const files = { drive: "server folder", whatch: "read-only" }[app.id];
      if (files) {
        add("path", { class: "wire wire--files", d: `M760 ${y} L800 ${y}` });
        add("rect", { class: "box", x: 800, y: y - 24, width: 120, height: 48, rx: 14 });
        add("text", { x: 814, y: y - 2, "font-size": 12, "font-weight": 600 }, "Your files");
        add("text", { x: 814, y: y + 14, "font-size": 11, class: "mono" }, files);
      }
    });
    $("#map-view").replaceChildren(svg);
  };

  /* ---------- load ---------- */
  const load = async () => {
    let apps = [];
    try {
      const res = await fetch(REGISTRY_URL, { cache: "no-store" });
      apps = (await res.json()).apps || [];
    } catch (e) {
      $("#subtitle").textContent = "Could not read the list of apps (" + REGISTRY_URL + ").";
      return;
    }

    const suite = apps.filter((app) => !["dashboard", "system"].includes(app.id));
    const installedSuite = suite.filter((app) => app.url);
    $("#stat-installed").textContent = String(installedSuite.length);
    $("#stat-installed-hint").textContent = `of ${suite.length} suite apps · plus the dashboard`;

    const status = new Map();
    renderApps(apps, status);
    renderSwitcher(apps);
    renderMap(apps);

    const checks = apps.filter((app) => app.url && app.id !== "system");
    await Promise.all(
      checks.map(async (app) => {
        status.set(app.id, await checkOnline(app.url));
        renderApps(apps, status);
      }),
    );
    const online = checks.filter((app) => status.get(app.id)).length;
    $("#stat-online").textContent = `${online}/${checks.length}`;
    $("#stat-online-hint").textContent = online === checks.length ? "everything is running" : "something is off";
    $("#server-dot").className = `status-dot ${online === checks.length ? "ok" : "off"}`;
    $("#server-status").textContent = online === checks.length ? "local server · online" : "local server · check apps";
  };

  load();
})();
